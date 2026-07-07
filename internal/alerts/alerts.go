package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type hubLike interface {
	core.App
	MakeLink(parts ...string) string
}

type AlertManager struct {
	hub           hubLike
	stopOnce      sync.Once
	pendingAlerts sync.Map
	alertsCache   *AlertsCache
}

type AlertMessageData struct {
	UserID   string
	SystemID string
	Title    string
	Message  string
	Link     string
	LinkText string
}

type SystemAlertFsStats struct {
	DiskTotal float64 `json:"d"`
	DiskUsed  float64 `json:"du"`
}

type SystemAlertStats struct {
	Cpu       float64                       `json:"cpu"`
	Mem       float64                       `json:"mp"`
	Disk      float64                       `json:"dp"`
	Bandwidth [2]uint64                     `json:"b"`
	ExtraFs   map[string]SystemAlertFsStats `json:"efs"`
}

type SystemAlertData struct {
	systemRecord *core.Record
	alertData    CachedAlertData
	name         string
	unit         string
	val          float64
	threshold    float64
	triggered    bool
	time         time.Time
	count        uint8
	min          uint8
	mapSums      map[string]float32
	descriptor   string
}

// NewAlertManager creates a new AlertManager instance.
func NewAlertManager(app hubLike) *AlertManager {
	am := &AlertManager{
		hub:         app,
		alertsCache: NewAlertsCache(app),
	}
	am.bindEvents()
	return am
}

func (am *AlertManager) bindEvents() {
	am.hub.OnServe().BindFunc(func(e *core.ServeEvent) error {
		_ = am.alertsCache.PopulateFromDB(true)
		if err := am.restorePendingStatusAlerts(); err != nil {
			e.App.Logger().Error("Failed to restore pending status alerts", "err", err)
		}
		return e.Next()
	})
}

// SendAlert sends an alert to the user via Telegram Bot API
func (am *AlertManager) SendAlert(data AlertMessageData) error {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if botToken == "" || chatID == "" {
		am.hub.Logger().Warn("Telegram alerts not configured. Set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID env vars.")
		return nil
	}

	text := fmt.Sprintf("⚠️ *%s*\n\n%s\n\n[Link](%s)", data.Title, data.Message, data.Link)

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		am.hub.Logger().Error("Failed to send Telegram alert", "err", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		am.hub.Logger().Error("Telegram API returned non-OK status", "status", resp.Status)
		return fmt.Errorf("telegram API error: %s", resp.Status)
	}

	am.hub.Logger().Info("Telegram alert sent successfully", "title", data.Title)
	return nil
}

func (am *AlertManager) setAlertTriggered(alert CachedAlertData, triggered bool) error {
	alertRecord, err := am.hub.FindRecordById("alerts", alert.Id)
	if err != nil {
		return err
	}
	alertRecord.Set("triggered", triggered)
	return am.hub.Save(alertRecord)
}
