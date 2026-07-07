package hub

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/henrygd/beszel"
	"github.com/henrygd/beszel/internal/hub/utils"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

var containerIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{12,64}$`)

// registerMiddlewares registers custom middlewares
func (h *Hub) registerMiddlewares(se *core.ServeEvent) {
	// authorizes request with user matching the provided email
	authorizeRequestWithEmail := func(e *core.RequestEvent, email string) (err error) {
		if e.Auth != nil || email == "" {
			return e.Next()
		}
		isAuthRefresh := e.Request.URL.Path == "/api/collections/users/auth-refresh" && e.Request.Method == http.MethodPost
		e.Auth, err = e.App.FindAuthRecordByEmail("users", email)
		if err != nil || !isAuthRefresh {
			return e.Next()
		}
		// auth refresh endpoint, make sure token is set in header
		token, _ := e.Auth.NewAuthToken()
		e.Request.Header.Set("Authorization", token)
		return e.Next()
	}
	// authenticate with auto login
	if autoLogin, _ := utils.GetEnv("AUTO_LOGIN"); autoLogin != "" {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return authorizeRequestWithEmail(e, autoLogin)
		})
	}
	// authenticate with trusted header
	if trustedHeader, _ := utils.GetEnv("TRUSTED_AUTH_HEADER"); trustedHeader != "" {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return authorizeRequestWithEmail(e, e.Request.Header.Get(trustedHeader))
		})
	}
}

// registerApiRoutes registers custom API routes
func (h *Hub) registerApiRoutes(se *core.ServeEvent) error {
	// auth protected routes
	apiAuth := se.Router.Group("/api/beszel")
	apiAuth.Bind(apis.RequireAuth())
	// auth optional routes
	apiNoAuth := se.Router.Group("/api/beszel")

	// create first user endpoint only needed if no users exist
	// Note: CreateFirstUser is defined in users package, but since we removed users package, we will define it.
	// Wait, we removed internal/users folder. We'll define a simple CreateFirstUser function directly or keep a simple handler.
	// Let's define the first-time admin setup handler directly in api.go or hub.go!
	apiNoAuth.POST("/create-user", h.CreateFirstUser)

	// check if first time setup on login page
	apiNoAuth.GET("/first-run", func(e *core.RequestEvent) error {
		total, err := e.App.CountRecords("users")
		return e.JSON(http.StatusOK, map[string]bool{"firstRun": err == nil && total == 0})
	})
	// get public key and version
	apiAuth.GET("/info", h.getInfo)
	apiAuth.GET("/getkey", h.getInfo) // deprecated - keep for compatibility w/ integrations
	
	// handle agent websocket connection
	apiNoAuth.GET("/agent-connect", h.handleAgentConnect)

	// Perform action on container
	apiAuth.POST("/containers/action", h.handleContainerAction)
	
	return nil
}

// getInfo returns data needed by authenticated users, such as the public key and current version
func (h *Hub) getInfo(e *core.RequestEvent) error {
	type infoResponse struct {
		Key         string `json:"key"`
		Version     string `json:"v"`
		CheckUpdate bool   `json:"cu"`
	}
	info := infoResponse{
		Key:     h.pubKey,
		Version: beszel.Version,
	}
	return e.JSON(http.StatusOK, info)
}

// CreateFirstUser handles the first user registration
func (h *Hub) CreateFirstUser(e *core.RequestEvent) error {
	total, err := e.App.CountRecords("users")
	if err != nil {
		return e.InternalServerError("Failed to check user count", err)
	}
	if total > 0 {
		return e.ForbiddenError("Registration is already complete", nil)
	}

	var data struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		PasswordConfirm string `json:"passwordConfirm"`
	}
	if err := e.BindBody(&data); err != nil {
		return e.BadRequestError("Invalid request payload", err)
	}

	collection, err := e.App.FindCachedCollectionByNameOrId("users")
	if err != nil {
		return e.InternalServerError("Failed to find users collection", err)
	}

	user := core.NewRecord(collection)
	user.SetEmail(data.Email)
	user.SetPassword(data.Password)
	user.Set("role", "admin") // First user is automatically admin

	if err := e.App.Save(user); err != nil {
		return e.BadRequestError("Failed to save user", err)
	}

	// Create default settings record for the user
	settingsCol, err := e.App.FindCachedCollectionByNameOrId("user_settings")
	if err == nil {
		settings := core.NewRecord(settingsCol)
		settings.Set("user", user.Id)
		settings.Set("emails", []string{})
		settings.Set("webhooks", []string{})
		_ = e.App.Save(settings)
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Hub) handleContainerAction(e *core.RequestEvent) error {
	var data struct {
		SystemID    string `json:"system"`
		ContainerID string `json:"container"`
		Action      string `json:"action"`
	}
	if err := e.BindBody(&data); err != nil {
		return e.BadRequestError("Invalid request payload", err)
	}

	if data.SystemID == "" || data.ContainerID == "" || data.Action == "" {
		return e.BadRequestError("Missing system, container, or action parameter", nil)
	}

	if !containerIDPattern.MatchString(data.ContainerID) {
		return e.BadRequestError("Invalid container ID format", nil)
	}

	system, err := h.sm.GetSystem(data.SystemID)
	if err != nil {
		return e.NotFoundError("System not found", err)
	}

	if system.WsConn == nil || !system.WsConn.IsConnected() {
		return e.BadRequestError("Agent disconnected", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errMsg, err := system.WsConn.ContainerAction(ctx, data.ContainerID, data.Action)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}

	if errMsg != "" {
		return e.BadRequestError(errMsg, nil)
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
