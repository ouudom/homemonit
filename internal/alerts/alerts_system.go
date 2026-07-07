package alerts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func (am *AlertManager) HandleSystemAlerts(systemRecord *core.Record, data *system.CombinedData) error {
	alerts := am.alertsCache.GetAlertsExcludingNames(systemRecord.Id, "Status")
	if len(alerts) == 0 {
		return nil
	}

	var validAlerts []SystemAlertData
	now := systemRecord.GetDateTime("updated").Time().UTC()
	oldestTime := now

	for _, alertData := range alerts {
		name := alertData.Name
		var val float64
		unit := "%"

		switch name {
		case "CPU":
			val = data.Info.Cpu
		case "Memory":
			val = data.Info.MemPct
		case "Disk":
			maxUsedPct := data.Info.DiskPct
			for _, fs := range data.Stats.ExtraFs {
				usedPct := fs.DiskUsed / fs.DiskTotal * 100
				if usedPct > maxUsedPct {
					maxUsedPct = usedPct
				}
			}
			val = maxUsedPct
		default:
			// Strip all other alerts
			continue
		}

		triggered := alertData.Triggered
		threshold := alertData.Value

		if (!triggered && val <= threshold) || (triggered && val > threshold) {
			continue
		}

		min := max(1, alertData.Min)

		alert := SystemAlertData{
			systemRecord: systemRecord,
			alertData:    alertData,
			name:         name,
			unit:         unit,
			val:          val,
			threshold:    threshold,
			triggered:    triggered,
			min:          min,
		}

		if min == 1 {
			alert.triggered = val > threshold
			go am.sendSystemAlert(alert)
			continue
		}

		alert.time = now.Add(-time.Duration(min) * time.Minute)
		if alert.time.Before(oldestTime) {
			oldestTime = alert.time
		}

		validAlerts = append(validAlerts, alert)
	}

	systemStats := []struct {
		Stats   []byte         `db:"stats"`
		Created types.DateTime `db:"created"`
	}{}

	err := am.hub.DB().
		Select("stats", "created").
		From("system_stats").
		Where(dbx.NewExp(
			"system={:system} AND type='1m' AND created > {:created}",
			dbx.Params{
				"system":  systemRecord.Id,
				"created": oldestTime.Add(-time.Second * 90),
			},
		)).
		OrderBy("created").
		All(&systemStats)
	if err != nil || len(systemStats) == 0 {
		return err
	}

	oldestRecordTime := systemStats[0].Created.Time()

	filteredAlerts := make([]SystemAlertData, 0, len(validAlerts))
	for _, alert := range validAlerts {
		if alert.time.After(oldestRecordTime) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}
	validAlerts = filteredAlerts

	if len(validAlerts) == 0 {
		return nil
	}

	var stats SystemAlertStats

	for i := range systemStats {
		stat := systemStats[i]
		systemStatsCreation := stat.Created.Time().Add(-time.Second * 10)
		if err := json.Unmarshal(stat.Stats, &stats); err != nil {
			return err
		}
		for j := range validAlerts {
			alert := &validAlerts[j]
			if i == 0 {
				alert.val = 0
			}
			if systemStatsCreation.Before(alert.time) {
				continue
			}
			switch alert.name {
			case "CPU":
				alert.val += stats.Cpu
			case "Memory":
				alert.val += stats.Mem
			case "Disk":
				if alert.mapSums == nil {
					alert.mapSums = make(map[string]float32, len(stats.ExtraFs)+1)
				}
				if _, ok := alert.mapSums["root"]; !ok {
					alert.mapSums["root"] = 0.0
				}
				alert.mapSums["root"] += float32(stats.Disk)
				for key, fs := range stats.ExtraFs {
					if fs.DiskTotal > 0 {
						if _, ok := alert.mapSums[key]; !ok {
							alert.mapSums[key] = 0.0
						}
						alert.mapSums[key] += float32(fs.DiskUsed / fs.DiskTotal * 100)
					}
				}
			default:
				continue
			}
			alert.count++
		}
	}

	for _, alert := range validAlerts {
		switch alert.name {
		case "Disk":
			maxPct := float32(0)
			for key, value := range alert.mapSums {
				sumPct := float32(value)
				if sumPct > maxPct {
					maxPct = sumPct
					alert.descriptor = fmt.Sprintf("Usage of %s", key)
				}
			}
			alert.val = float64(maxPct / float32(alert.count))
		default:
			alert.val = alert.val / float64(alert.count)
		}
		minCount := float32(alert.min) / 1.2
		if float32(alert.count) >= minCount {
			if !alert.triggered && alert.val > alert.threshold {
				alert.triggered = true
				go am.sendSystemAlert(alert)
			} else if alert.triggered && alert.val <= alert.threshold {
				alert.triggered = false
				go am.sendSystemAlert(alert)
			}
		}
	}
	return nil
}

func (am *AlertManager) sendSystemAlert(alert SystemAlertData) {
	systemName := alert.systemRecord.GetString("name")

	if alert.name == "Disk" {
		alert.name += " usage"
	}

	titleAlertName := alert.name
	if titleAlertName != "CPU" {
		titleAlertName = strings.ToLower(titleAlertName)
	}

	var subject string
	if alert.triggered {
		subject = fmt.Sprintf("%s %s above threshold", systemName, titleAlertName)
	} else {
		subject = fmt.Sprintf("%s %s below threshold", systemName, titleAlertName)
	}
	minutesLabel := "minute"
	if alert.min > 1 {
		minutesLabel += "s"
	}
	if alert.descriptor == "" {
		alert.descriptor = alert.name
	}
	body := fmt.Sprintf("%s averaged %.2f%s for the previous %v %s.", alert.descriptor, alert.val, alert.unit, alert.min, minutesLabel)

	if err := am.setAlertTriggered(alert.alertData, alert.triggered); err != nil {
		return
	}
	am.SendAlert(AlertMessageData{
		UserID:   alert.alertData.UserID,
		SystemID: alert.systemRecord.Id,
		Title:    subject,
		Message:  body,
		Link:     am.hub.MakeLink("system", alert.systemRecord.Id),
		LinkText: "View " + systemName,
	})
}
