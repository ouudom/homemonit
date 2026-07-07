package hub

import (
	"net/http"

	"github.com/blang/semver"
	"github.com/lxzan/gws"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/henrygd/beszel/internal/hub/ws"
)

// agentConnectRequest holds information related to an agent's connection attempt.
type agentConnectRequest struct {
	hub         *Hub
	req         *http.Request
	res         http.ResponseWriter
	token       string
	agentSemVer semver.Version
}

// handleAgentConnect is the HTTP handler for an agent's connection request.
func (h *Hub) handleAgentConnect(e *core.RequestEvent) error {
	agentRequest := agentConnectRequest{req: e.Request, res: e.Response, hub: h}
	_ = agentRequest.agentConnect()
	return nil
}

// agentConnect validates agent credentials and upgrades the connection to a WebSocket.
func (acr *agentConnectRequest) agentConnect() (err error) {
	var agentVersion string

	acr.token, agentVersion, err = acr.validateAgentHeaders(acr.req.Header)
	if err != nil {
		return acr.sendResponseError(acr.res, http.StatusBadRequest, "")
	}

	// Validate agent version
	acr.agentSemVer, err = semver.Parse(agentVersion)
	if err != nil {
		return acr.sendResponseError(acr.res, http.StatusUnauthorized, "Invalid agent version")
	}

	// Find matching system record where the port field is used to store the authentication token
	systemRecord, err := acr.hub.FindFirstRecordByFilter("systems", "port = {:token}", dbx.Params{"token": acr.token})
	if err != nil {
		return acr.sendResponseError(acr.res, http.StatusUnauthorized, "Invalid token")
	}

	// Upgrade connection to WebSocket
	conn, err := ws.GetUpgrader().Upgrade(acr.res, acr.req)
	if err != nil {
		return acr.sendResponseError(acr.res, http.StatusInternalServerError, "WebSocket upgrade failed")
	}

	go acr.verifyWsConn(conn, systemRecord)

	return nil
}

// verifyWsConn sets up the WebSocket connection and registers the system in the system manager.
func (acr *agentConnectRequest) verifyWsConn(conn *gws.Conn, systemRecord *core.Record) error {
	wsConn := ws.NewWsConnection(conn, acr.agentSemVer)
	conn.Session().Store("wsConn", wsConn)

	go conn.ReadLoop()

	return acr.hub.sm.AddWebSocketSystem(systemRecord.Id, acr.agentSemVer, wsConn)
}

// validateAgentHeaders extracts and validates the token and agent version from HTTP headers.
func (acr *agentConnectRequest) validateAgentHeaders(headers http.Header) (string, string, error) {
	token := headers.Get("X-Token")
	agentVersion := headers.Get("X-Beszel")

	if agentVersion == "" || token == "" || len(token) > 64 {
		return "", "", http.ErrBodyNotAllowed
	}
	return token, agentVersion, nil
}

// sendResponseError writes an HTTP error response.
func (acr *agentConnectRequest) sendResponseError(res http.ResponseWriter, code int, message string) error {
	res.WriteHeader(code)
	if message != "" {
		res.Write([]byte(message))
	}
	return nil
}
