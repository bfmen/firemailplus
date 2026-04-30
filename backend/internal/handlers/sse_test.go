package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"firemail/internal/auth"
	"firemail/internal/config"
	"firemail/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSSEHeadersDisableProxyBuffering(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))

	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "sse-handler-test-secret",
			JWTExpiry: time.Minute,
		},
		Database: config.DatabaseConfig{
			Path:                "file:" + t.Name() + "?mode=memory&cache=shared",
			BackupDir:           t.TempDir(),
			BackupMaxCount:      3,
			BackupIntervalHours: 24,
		},
		SSE: config.SSEConfig{
			MaxConnectionsPerUser: 5,
			ConnectionTimeout:     time.Minute,
			HeartbeatInterval:     time.Second,
			CleanupInterval:       time.Minute,
			BufferSize:            16,
			EnableHeartbeat:       false,
		},
	}

	user := &models.User{
		Username:    "sse-user",
		Password:    "sse-password",
		DisplayName: "SSE User",
		Email:       "sse-user@example.test",
		Role:        "user",
		IsActive:    true,
	}
	require.NoError(t, db.Create(user).Error)

	authService := auth.NewService(db, auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry))
	login, err := authService.Login(&auth.LoginRequest{Username: "sse-user", Password: "sse-password"})
	require.NoError(t, err)

	handler := New(db, cfg)
	router := gin.New()
	router.GET("/api/v1/sse/events", handler.HandleSSE)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(20*time.Millisecond, cancel)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/sse/events?client_id=test-client&token="+url.QueryEscape(login.Token),
		nil,
	).WithContext(ctx)
	req.Header.Set("Accept", "text/event-stream")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	require.Equal(t, "no-cache, no-transform", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "keep-alive", recorder.Header().Get("Connection"))
	require.Equal(t, "no", recorder.Header().Get("X-Accel-Buffering"))
}
