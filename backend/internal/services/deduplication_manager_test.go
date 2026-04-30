package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

func TestDeduplicationScheduleAppliesDefaults(t *testing.T) {
	manager := NewDeduplicationManager(nil, nil, nil).(*StandardDeduplicationManager)

	schedule := &DeduplicationSchedule{Enabled: true}
	require.NoError(t, manager.ScheduleDeduplication(context.Background(), 1, schedule))
	require.Equal(t, DefaultDeduplicationScheduleFrequency, schedule.Frequency)
	require.Equal(t, DefaultDeduplicationScheduleTime, schedule.Time)
	require.False(t, schedule.NextRun.IsZero())
}

func TestDeduplicationReportFallsBackWithoutEnhancedDeduplicator(t *testing.T) {
	previousEnhanced := config.Env.EnableEnhancedDedup
	config.Env.EnableEnhancedDedup = false
	t.Cleanup(func() {
		config.Env.EnableEnhancedDedup = previousEnhanced
	})

	db := setupDeduplicationManagerTestDB(t)
	manager := NewDeduplicationManager(db, nil, nil).(*StandardDeduplicationManager)
	accountID := uint(42)

	createDeduplicationManagerEmail(t, db, accountID, "dup-a")
	createDeduplicationManagerEmail(t, db, accountID, "dup-a")
	createDeduplicationManagerEmail(t, db, accountID, "unique")
	createDeduplicationManagerEmail(t, db, 99, "dup-a")

	report, err := manager.GetDeduplicationReport(context.Background(), accountID)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.Stats)
	require.Equal(t, accountID, report.AccountID)
	require.Equal(t, int64(3), report.Stats.TotalChecked)
	require.Equal(t, int64(1), report.Stats.DuplicatesFound)
	require.NotZero(t, report.GeneratedAt)
}

func setupDeduplicationManagerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.EmailAccount{},
		&models.Folder{},
		&models.Email{},
		&DeduplicationActivity{},
	))
	return db
}

func createDeduplicationManagerEmail(t *testing.T, db *gorm.DB, accountID uint, messageID string) {
	t.Helper()

	email := &models.Email{
		AccountID: accountID,
		MessageID: fmt.Sprintf("<%s@example.test>", messageID),
		UID:       uint32(time.Now().UnixNano()),
		Subject:   messageID,
		Date:      time.Now().UTC(),
		From:      "sender@example.test",
		IsRead:    false,
		IsDeleted: false,
	}
	require.NoError(t, db.Create(email).Error)
}
