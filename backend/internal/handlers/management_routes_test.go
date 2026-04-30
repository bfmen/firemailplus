package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"firemail/internal/auth"
	"firemail/internal/config"
	"firemail/internal/middleware"
	"firemail/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestManagementRoutesRequireAdminRole(t *testing.T) {
	router, _, userToken, adminToken := setupManagementRouteTest(t)

	userReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/backups", nil)
	userReq.Header.Set("Authorization", "Bearer "+userToken)
	userResp := httptest.NewRecorder()
	router.ServeHTTP(userResp, userReq)
	require.Equal(t, http.StatusForbidden, userResp.Code)

	adminReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/backups", nil)
	adminReq.Header.Set("Authorization", "Bearer "+adminToken)
	adminResp := httptest.NewRecorder()
	router.ServeHTTP(adminResp, adminReq)
	require.Equal(t, http.StatusOK, adminResp.Code)
}

func TestAuthManagementRoutesAreRegistered(t *testing.T) {
	router, db, userToken, _ := setupManagementRouteTest(t)

	profileBody := bytes.NewBufferString(`{"display_name":"Updated User","email":"updated@example.test"}`)
	profileReq := httptest.NewRequest(http.MethodPut, "/api/v1/auth/profile", profileBody)
	profileReq.Header.Set("Authorization", "Bearer "+userToken)
	profileReq.Header.Set("Content-Type", "application/json")
	profileResp := httptest.NewRecorder()
	router.ServeHTTP(profileResp, profileReq)
	require.Equal(t, http.StatusOK, profileResp.Code)

	var updated models.User
	require.NoError(t, db.Where("username = ?", "normal-user").First(&updated).Error)
	require.Equal(t, "Updated User", updated.DisplayName)

	passwordBody := bytes.NewBufferString(`{"old_password":"old-password","new_password":"new-password"}`)
	passwordReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", passwordBody)
	passwordReq.Header.Set("Authorization", "Bearer "+userToken)
	passwordReq.Header.Set("Content-Type", "application/json")
	passwordResp := httptest.NewRecorder()
	router.ServeHTTP(passwordResp, passwordReq)
	require.Equal(t, http.StatusOK, passwordResp.Code)

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshReq.Header.Set("Authorization", "Bearer "+userToken)
	refreshResp := httptest.NewRecorder()
	router.ServeHTTP(refreshResp, refreshReq)
	require.Equal(t, http.StatusOK, refreshResp.Code)

	var refreshPayload SuccessResponse
	require.NoError(t, json.Unmarshal(refreshResp.Body.Bytes(), &refreshPayload))
	require.True(t, refreshPayload.Success)
}

func setupManagementRouteTest(t *testing.T) (*gin.Engine, *gorm.DB, string, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))

	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "management-route-test-secret",
			JWTExpiry: 5 * time.Minute,
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

	handler := New(db, cfg)
	router := gin.New()
	api := router.Group("/api/v1")
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/refresh", handler.AuthRequired(), handler.RefreshToken)
		authGroup.POST("/change-password", handler.AuthRequired(), handler.ChangePassword)
		authGroup.PUT("/profile", handler.AuthRequired(), handler.UpdateProfile)
	}
	admin := api.Group("/admin")
	admin.Use(handler.AuthRequired(), middleware.AdminRequired())
	{
		admin.GET("/backups", handler.ListBackups)
	}

	userToken := createManagementRouteUserAndToken(t, db, cfg, "normal-user", "user")
	adminToken := createManagementRouteUserAndToken(t, db, cfg, "admin-user", "admin")
	return router, db, userToken, adminToken
}

func createManagementRouteUserAndToken(t *testing.T, db *gorm.DB, cfg *config.Config, username, role string) string {
	t.Helper()

	user := &models.User{
		Username:    username,
		Password:    "old-password",
		DisplayName: username,
		Email:       username + "@example.test",
		Role:        role,
		IsActive:    true,
	}
	require.NoError(t, db.Create(user).Error)

	authService := auth.NewService(db, auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry))
	login, err := authService.Login(&auth.LoginRequest{Username: username, Password: "old-password"})
	require.NoError(t, err)
	return login.Token
}
