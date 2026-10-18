package service

import (
	"strings"
	"time"
)

// A job is late only after its next scheduled run and a grace period. Daily
// aggregation runs hourly; the cleanup cron can be daily, weekly, or custom.
func opsJobStaleAt(now time.Time, hb *OpsJobHeartbeat, cleanupSchedule string, loc *time.Location) bool {
	if hb == nil || hb.LastSuccessAt == nil {
		return false
	}
	last := *hb.LastSuccessAt
	deadline := last.Add(15 * time.Minute)
	switch hb.JobName {
	case opsAggHourlyJobName:
		deadline = last.Add(opsAggHourlyInterval + 15*time.Minute)
	case opsAggDailyJobName:
		deadline = last.Add(opsAggDailyInterval + 15*time.Minute)
	case opsCleanupJobName:
		schedule, err := opsCleanupCronParser.Parse(cleanupSchedule)
		if err != nil {
			return false // An unknown schedule is not evidence of a missed run.
		}
		deadline = schedule.Next(last.In(loc)).Add(30 * time.Minute)
	}
	return now.After(deadline)
}

func opsJobHeartbeatStale(now time.Time, hb *OpsJobHeartbeat) bool {
	if hb.Stale != nil {
		return *hb.Stale
	}
	return opsJobStaleAt(now, hb, opsCleanupDefaultSchedule, time.Local)
}

func (s *OpsService) populateJobStaleness(now time.Time, jobs []*OpsJobHeartbeat) {
	if len(jobs) == 0 {
		return
	}
	settings := s.OpsAdvancedSettingsSnapshot()
	if s.runtimeSettings.Load() == nil && s.cfg != nil && strings.TrimSpace(s.cfg.Ops.Cleanup.Schedule) != "" {
		settings.DataRetention.CleanupSchedule = s.cfg.Ops.Cleanup.Schedule
	}
	loc := time.Local
	if s.cfg != nil && strings.TrimSpace(s.cfg.Timezone) != "" {
		if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil {
			loc = parsed
		}
	}
	for _, hb := range jobs {
		if hb == nil {
			continue
		}
		stale := opsJobStaleAt(now, hb, settings.DataRetention.CleanupSchedule, loc)
		if hb.JobName == opsCleanupJobName && !settings.DataRetention.CleanupEnabled {
			stale = false
		}
		hb.Stale = &stale
	}
}
