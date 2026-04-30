package migration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type forceTrackingMigrator struct {
	forceCalled bool
}

func (m *forceTrackingMigrator) Up(context.Context) error                   { return nil }
func (m *forceTrackingMigrator) Down(context.Context) error                 { return nil }
func (m *forceTrackingMigrator) Steps(context.Context, int) error           { return nil }
func (m *forceTrackingMigrator) Force(context.Context, int) error           { m.forceCalled = true; return nil }
func (m *forceTrackingMigrator) Version(context.Context) (int, bool, error) { return 0, false, nil }
func (m *forceTrackingMigrator) Close() error                               { return nil }

func TestRecoverFromDirtyStateRefusesAutomaticForce(t *testing.T) {
	migrator := &forceTrackingMigrator{}
	service := NewMigrationService(nil)
	service.migrator = migrator

	err := service.recoverFromDirtyState(context.Background(), 8)

	require.ErrorContains(t, err, "manual repair is required")
	require.False(t, migrator.forceCalled)
}
