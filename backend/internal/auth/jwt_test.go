package auth

import (
	"testing"
	"time"

	"firemail/internal/models"

	"github.com/stretchr/testify/require"
)

func TestRefreshTokenAllowsFreshValidToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	user := &models.User{
		BaseModel: models.BaseModel{ID: 42},
		Username:  "fresh-user",
		Role:      "user",
	}

	token, err := manager.GenerateToken(user)
	require.NoError(t, err)

	refreshed, err := manager.RefreshToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, refreshed)

	claims, err := manager.ValidateToken(refreshed)
	require.NoError(t, err)
	require.Equal(t, user.ID, claims.UserID)
	require.Equal(t, user.Username, claims.Username)
	require.Equal(t, user.Role, claims.Role)
	require.Greater(t, time.Until(claims.ExpiresAt.Time), 50*time.Minute)
}

func TestRefreshTokenRejectsExpiredToken(t *testing.T) {
	manager := NewJWTManager("test-secret", -time.Minute)
	user := &models.User{
		BaseModel: models.BaseModel{ID: 7},
		Username:  "expired-user",
		Role:      "user",
	}

	token, err := manager.GenerateToken(user)
	require.NoError(t, err)

	refreshed, err := manager.RefreshToken(token)
	require.Error(t, err)
	require.Empty(t, refreshed)
}
