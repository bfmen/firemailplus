package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminCredentialsRejectMissingProductionPassword(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("ADMIN_USERNAME", "")
	t.Setenv("ADMIN_PASSWORD", "")

	_, _, err := adminCredentialsFromEnv()
	require.ErrorContains(t, err, "ADMIN_PASSWORD must be set in production")
}

func TestAdminCredentialsRejectWeakProductionPassword(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("ADMIN_PASSWORD", "admin123")

	_, _, err := adminCredentialsFromEnv()
	require.ErrorContains(t, err, "weak default")
}

func TestAdminCredentialsAllowDevelopmentDefault(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("ADMIN_USERNAME", "")
	t.Setenv("ADMIN_PASSWORD", "")

	username, password, err := adminCredentialsFromEnv()
	require.NoError(t, err)
	require.Equal(t, "admin", username)
	require.Equal(t, "admin123", password)
}
