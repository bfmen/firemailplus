package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeduplicationScheduleAndCancelAreStateful(t *testing.T) {
	manager := NewDeduplicationManager(nil, nil, nil).(*StandardDeduplicationManager)

	schedule := &DeduplicationSchedule{
		Enabled:   true,
		Frequency: "daily",
		Time:      time.Now().Add(time.Hour).Format("15:04"),
		Options:   &DeduplicationOptions{DryRun: true},
	}
	require.NoError(t, manager.ScheduleDeduplication(context.Background(), 42, schedule))
	require.False(t, schedule.NextRun.IsZero())

	manager.schedulesMu.RLock()
	stored := manager.schedules[42]
	manager.schedulesMu.RUnlock()
	require.Same(t, schedule, stored)

	require.NoError(t, manager.CancelScheduledDeduplication(context.Background(), 42))
	manager.schedulesMu.RLock()
	_, exists := manager.schedules[42]
	manager.schedulesMu.RUnlock()
	require.False(t, exists)
}

func TestDeduplicationScheduleRejectsInvalidInput(t *testing.T) {
	manager := NewDeduplicationManager(nil, nil, nil).(*StandardDeduplicationManager)

	require.Error(t, manager.ScheduleDeduplication(context.Background(), 1, nil))
	require.Error(t, manager.ScheduleDeduplication(context.Background(), 1, &DeduplicationSchedule{
		Enabled:   true,
		Frequency: "hourly",
		Time:      "12:00",
	}))
	require.Error(t, manager.ScheduleDeduplication(context.Background(), 1, &DeduplicationSchedule{
		Enabled:   true,
		Frequency: "daily",
		Time:      "99:00",
	}))
}
