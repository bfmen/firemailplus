package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"firemail/internal/models"
	"firemail/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeduplicationHandlerRejectsCrossUserAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupDeduplicationHandlerTestDB(t)
	owner := createDeduplicationHandlerUser(t, db, "owner")
	attacker := createDeduplicationHandlerUser(t, db, "attacker")
	account := createDeduplicationHandlerAccount(t, db, owner.ID)
	manager := &deduplicationHandlerFakeManager{}
	handler := NewDeduplicationHandler(manager, db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/deduplication/accounts/1/schedule", bytes.NewBufferString(`{"enabled":true,"frequency":"daily","time":"12:00"}`))
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("userID", attacker.ID)

	handler.ScheduleDeduplication(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.False(t, manager.scheduleCalled)
	require.Error(t, handler.validateAccountAccess(c, account.ID, attacker.ID))
	require.NoError(t, handler.validateAccountAccess(c, account.ID, owner.ID))
}

func setupDeduplicationHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.EmailAccount{}))
	return db
}

func createDeduplicationHandlerUser(t *testing.T, db *gorm.DB, username string) *models.User {
	t.Helper()

	user := &models.User{
		Username:    username,
		Password:    "password123",
		DisplayName: username,
		Role:        "user",
		IsActive:    true,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createDeduplicationHandlerAccount(t *testing.T, db *gorm.DB, userID uint) *models.EmailAccount {
	t.Helper()

	account := &models.EmailAccount{
		UserID:       userID,
		Name:         "Dedup Account",
		Email:        "dedup@example.test",
		Provider:     "custom",
		AuthMethod:   "password",
		Username:     "dedup@example.test",
		IMAPHost:     "imap.example.test",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.example.test",
		SMTPPort:     587,
		SMTPSecurity: "STARTTLS",
		IsActive:     true,
	}
	require.NoError(t, db.Create(account).Error)
	return account
}

type deduplicationHandlerFakeManager struct {
	scheduleCalled bool
}

func (m *deduplicationHandlerFakeManager) DeduplicateAccount(context.Context, uint, *services.DeduplicationOptions) (*services.BatchDeduplicationResult, error) {
	return &services.BatchDeduplicationResult{}, nil
}
func (m *deduplicationHandlerFakeManager) DeduplicateUser(context.Context, uint, *services.DeduplicationOptions) (*services.UserDeduplicationResult, error) {
	return &services.UserDeduplicationResult{}, nil
}
func (m *deduplicationHandlerFakeManager) GetDeduplicationReport(context.Context, uint) (*services.DeduplicationReport, error) {
	return &services.DeduplicationReport{}, nil
}
func (m *deduplicationHandlerFakeManager) ScheduleDeduplication(context.Context, uint, *services.DeduplicationSchedule) error {
	m.scheduleCalled = true
	return nil
}
func (m *deduplicationHandlerFakeManager) CancelScheduledDeduplication(context.Context, uint) error {
	return nil
}
