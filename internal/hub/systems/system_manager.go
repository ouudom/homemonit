package systems

import (
	"errors"
	"fmt"
	"time"

	"github.com/henrygd/beszel/internal/hub/ws"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/common"

	"github.com/blang/semver"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/store"
)

const (
	up      string = "up"
	down    string = "down"
	paused  string = "paused"
	pending string = "pending"
	interval int = 60_000
)

var errSystemExists = errors.New("system exists")

type SystemManager struct {
	hub     hubLike
	systems *store.Store[string, *System]
}

type hubLike interface {
	core.App
	HandleSystemAlerts(systemRecord *core.Record, data *system.CombinedData) error
	HandleStatusAlerts(status string, systemRecord *core.Record) error
	CancelPendingStatusAlerts(systemID string)
}

func NewSystemManager(hub hubLike) *SystemManager {
	return &SystemManager{
		systems: store.New(map[string]*System{}),
		hub:     hub,
	}
}

func (sm *SystemManager) GetSystem(systemID string) (*System, error) {
	sys, ok := sm.systems.GetOk(systemID)
	if !ok {
		return nil, fmt.Errorf("system not found")
	}
	return sys, nil
}

func (sm *SystemManager) Initialize() error {
	sm.bindEventHooks()

	var systems []*System
	err := sm.hub.DB().NewQuery("SELECT id, host, port, status FROM systems WHERE status != 'paused'").All(&systems)
	if err != nil || len(systems) == 0 {
		return err
	}

	go func() {
		for _, system := range systems {
			_ = sm.AddSystem(system)
		}
	}()
	return nil
}

func (sm *SystemManager) bindEventHooks() {
	sm.hub.OnRecordCreate("systems").BindFunc(sm.onRecordCreate)
	sm.hub.OnRecordAfterCreateSuccess("systems").BindFunc(sm.onRecordAfterCreateSuccess)
	sm.hub.OnRecordUpdate("systems").BindFunc(sm.onRecordUpdate)
	sm.hub.OnRecordAfterUpdateSuccess("systems").BindFunc(sm.onRecordAfterUpdateSuccess)
	sm.hub.OnRecordAfterDeleteSuccess("systems").BindFunc(sm.onRecordAfterDeleteSuccess)
	sm.hub.OnRealtimeSubscribeRequest().BindFunc(sm.onRealtimeSubscribeRequest)
	sm.hub.OnRealtimeConnectRequest().BindFunc(sm.onRealtimeConnectRequest)
}

func (sm *SystemManager) onRecordCreate(e *core.RecordEvent) error {
	e.Record.Set("info", system.Info{})
	e.Record.Set("status", pending)
	return e.Next()
}

func (sm *SystemManager) onRecordAfterCreateSuccess(e *core.RecordEvent) error {
	if err := sm.AddRecord(e.Record, nil); err != nil {
		e.App.Logger().Error("Error adding record", "err", err)
	}
	return e.Next()
}

func (sm *SystemManager) onRecordUpdate(e *core.RecordEvent) error {
	if e.Record.GetString("status") == paused {
		e.Record.Set("info", system.Info{})
	}
	return e.Next()
}

func (sm *SystemManager) onRecordAfterUpdateSuccess(e *core.RecordEvent) error {
	newStatus := e.Record.GetString("status")
	prevStatus := pending
	system, ok := sm.systems.GetOk(e.Record.Id)
	if ok {
		prevStatus = system.Status
		system.Status = newStatus
	}

	switch newStatus {
	case paused:
		_ = deactivateAlerts(e.App, e.Record.Id)
		sm.hub.CancelPendingStatusAlerts(e.Record.Id)
		return e.Next()
	case pending:
		if ok && system.WsConn != nil {
			go system.update()
			return e.Next()
		}
		if err := sm.AddRecord(e.Record, nil); err != nil {
			e.App.Logger().Error("Error adding record", "err", err)
		}
		_ = deactivateAlerts(e.App, e.Record.Id)
		return e.Next()
	}

	if !ok {
		return sm.AddRecord(e.Record, nil)
	}

	if newStatus == up {
		if err := sm.hub.HandleSystemAlerts(e.Record, system.data); err != nil {
			e.App.Logger().Error("Error handling system alerts", "err", err)
		}
	}

	if (newStatus == down && prevStatus == up) || (newStatus == up && prevStatus == down) {
		if err := sm.hub.HandleStatusAlerts(newStatus, e.Record); err != nil {
			e.App.Logger().Error("Error handling status alerts", "err", err)
		}
	}
	return e.Next()
}

func (sm *SystemManager) onRecordAfterDeleteSuccess(e *core.RecordEvent) error {
	sm.RemoveSystem(e.Record.Id)
	return e.Next()
}

func (sm *SystemManager) AddSystem(sys *System) error {
	if sm.systems.Has(sys.Id) {
		return errSystemExists
	}
	if sys.Id == "" || sys.Host == "" {
		return errors.New("system missing required fields")
	}

	sys.manager = sm
	sys.ctx, sys.cancel = sys.getContext()
	sys.data = &system.CombinedData{}
	sm.systems.Set(sys.Id, sys)

	go sys.StartUpdater()
	return nil
}

func (sm *SystemManager) RemoveSystem(systemID string) error {
	system, ok := sm.systems.GetOk(systemID)
	if !ok {
		return errors.New("system not found")
	}

	if system.cancel != nil {
		system.cancel()
	}

	system.closeWebSocketConnection()
	sm.systems.Remove(systemID)
	return nil
}

func (sm *SystemManager) AddRecord(record *core.Record, system *System) (err error) {
	if sm.systems.Has(record.Id) {
		_ = sm.RemoveSystem(record.Id)
	}

	if system == nil {
		system = sm.NewSystem(record.Id)
	}

	system.Status = record.GetString("status")
	system.Host = record.GetString("host")
	system.Port = record.GetString("port")

	return sm.AddSystem(system)
}

func (sm *SystemManager) AddWebSocketSystem(systemId string, agentVersion semver.Version, wsConn *ws.WsConn) error {
	systemRecord, err := sm.hub.FindRecordById("systems", systemId)
	if err != nil {
		return err
	}

	system := sm.NewSystem(systemId)
	system.WsConn = wsConn
	system.agentVersion = agentVersion

	if err := sm.AddRecord(systemRecord, system); err != nil {
		return err
	}
	return nil
}

func deactivateAlerts(app core.App, systemID string) error {
	alerts, err := app.FindRecordsByFilter("alerts", fmt.Sprintf("system = '%s' && triggered = 1", systemID), "", -1, 0)
	if err != nil {
		return err
	}

	for _, alert := range alerts {
		alert.Set("triggered", false)
		if err := app.SaveNoValidate(alert); err != nil {
			return err
		}
	}
	return nil
}
