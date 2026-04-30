package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBatchMarkAccountsAsReadReturnsAcceptedJob(t *testing.T) {
	router, db, token := setupEmailAccountHandlerTest(t)

	var user models.User
	require.NoError(t, db.Where("username = ?", "account-handler-user").First(&user).Error)

	account := &models.EmailAccount{
		UserID:       user.ID,
		Name:         "handler@example.test",
		Email:        "handler@example.test",
		Provider:     "custom",
		AuthMethod:   "password",
		IMAPHost:     "imap.example.test",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		IsActive:     true,
		SyncStatus:   "pending",
	}
	require.NoError(t, db.Create(account).Error)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/batch/mark-read", strings.NewReader(`{"account_ids":[`+strconvUint(account.ID)+`]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			JobID          string `json:"job_id"`
			Status         string `json:"status"`
			ProcessedCount int    `json:"processed_count"`
			TotalCount     int    `json:"total_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.NotEmpty(t, response.Data.JobID)
	require.Equal(t, models.MailboxJobStatusQueued, response.Data.Status)
	require.Equal(t, 1, response.Data.TotalCount)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/batch/mark-read/"+response.Data.JobID, nil)
	statusReq.Header.Set("Authorization", "Bearer "+token)
	statusRecorder := httptest.NewRecorder()
	router.ServeHTTP(statusRecorder, statusReq)

	require.Equal(t, http.StatusOK, statusRecorder.Code)
}

func TestBatchMarkAccountsAsReadRejectsEmptyAccountIDs(t *testing.T) {
	router, _, token := setupEmailAccountHandlerTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/batch/mark-read", strings.NewReader(`{"account_ids":[]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func setupEmailAccountHandlerTest(t *testing.T) (*gin.Engine, *gorm.DB, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.EmailAccount{}, &models.MailboxJob{}))

	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "account-handler-test-secret",
			JWTExpiry: time.Hour,
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
	accounts := router.Group("/api/v1/accounts")
	accounts.Use(handler.AuthRequired())
	{
		accounts.POST("/batch/mark-read", handler.BatchMarkAccountsAsRead)
		accounts.GET("/batch/mark-read/:job_id", handler.GetAccountJobStatus)
	}

	token := createManagementRouteUserAndToken(t, db, cfg, "account-handler-user", "user")
	return router, db, token
}

func strconvUint(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
