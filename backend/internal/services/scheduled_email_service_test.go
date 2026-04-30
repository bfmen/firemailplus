package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"firemail/internal/models"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScheduledEmailServiceProcessesDueRetryAndMarksQueuedNotSent(t *testing.T) {
	db := setupScheduledEmailTestDB(t)
	past := time.Now().Add(-time.Minute)
	request := scheduledEmailTestRequest()
	emailData, err := json.Marshal(request)
	require.NoError(t, err)

	queue := &models.SendQueue{
		SendID:      "scheduled-retry",
		UserID:      1,
		AccountID:   2,
		EmailData:   string(emailData),
		ScheduledAt: &past,
		Status:      "retry",
		Attempts:    1,
		MaxAttempts: 3,
		NextAttempt: &past,
	}
	require.NoError(t, db.Create(queue).Error)

	sender := &scheduledEmailTestSender{result: &SendResult{SendID: "actual-send", Status: "pending"}}
	service := NewScheduledEmailService(db, nil, &scheduledEmailTestComposer{}, sender)

	require.NoError(t, service.ProcessScheduledEmails(context.Background()))

	var updated models.SendQueue
	require.NoError(t, db.Where("send_id = ?", "scheduled-retry").First(&updated).Error)
	require.Equal(t, "queued", updated.Status)
	require.Equal(t, 2, updated.Attempts)
	require.Nil(t, updated.NextAttempt)
	require.Empty(t, updated.LastError)
	require.Equal(t, uint(2), sender.accountID)
	require.Len(t, sender.emails, 1)
}

func TestScheduledEmailServiceSchedulesRetryOnFailure(t *testing.T) {
	db := setupScheduledEmailTestDB(t)
	past := time.Now().Add(-time.Minute)
	request := scheduledEmailTestRequest()
	emailData, err := json.Marshal(request)
	require.NoError(t, err)

	queue := &models.SendQueue{
		SendID:      "scheduled-fail",
		UserID:      1,
		AccountID:   2,
		EmailData:   string(emailData),
		ScheduledAt: &past,
		Status:      "scheduled",
		Attempts:    0,
		MaxAttempts: 3,
	}
	require.NoError(t, db.Create(queue).Error)

	service := NewScheduledEmailService(db, nil, &scheduledEmailTestComposer{err: errors.New("compose failed")}, &scheduledEmailTestSender{})

	require.NoError(t, service.ProcessScheduledEmails(context.Background()))

	var updated models.SendQueue
	require.NoError(t, db.Where("send_id = ?", "scheduled-fail").First(&updated).Error)
	require.Equal(t, "retry", updated.Status)
	require.Equal(t, 1, updated.Attempts)
	require.Contains(t, updated.LastError, "compose failed")
	require.NotNil(t, updated.NextAttempt)
	require.True(t, updated.NextAttempt.After(time.Now()))
}

func setupScheduledEmailTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.SendQueue{}))
	return db
}

func scheduledEmailTestRequest() *ComposeEmailRequest {
	return &ComposeEmailRequest{
		From:     &models.EmailAddress{Name: "Sender", Address: "sender@example.test"},
		To:       []*models.EmailAddress{{Name: "Recipient", Address: "recipient@example.test"}},
		Subject:  "Scheduled",
		TextBody: "Scheduled body",
	}
}

type scheduledEmailTestComposer struct {
	err error
}

func (c *scheduledEmailTestComposer) ComposeEmail(context.Context, *ComposeEmailRequest) (*ComposedEmail, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &ComposedEmail{
		ID:       "composed-scheduled",
		From:     &models.EmailAddress{Name: "Sender", Address: "sender@example.test"},
		To:       []*models.EmailAddress{{Name: "Recipient", Address: "recipient@example.test"}},
		Subject:  "Scheduled",
		TextBody: "Scheduled body",
	}, nil
}
func (c *scheduledEmailTestComposer) ValidateEmail(*ComposedEmail) error { return nil }
func (c *scheduledEmailTestComposer) AddAttachment(*ComposedEmail, *EmailAttachment) error {
	return nil
}
func (c *scheduledEmailTestComposer) AddInlineAttachment(*ComposedEmail, *InlineAttachment) error {
	return nil
}

type scheduledEmailTestSender struct {
	result    *SendResult
	err       error
	accountID uint
	emails    []*ComposedEmail
}

func (s *scheduledEmailTestSender) SendEmail(_ context.Context, email *ComposedEmail, accountID uint) (*SendResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.accountID = accountID
	s.emails = append(s.emails, email)
	if s.result != nil {
		return s.result, nil
	}
	return &SendResult{SendID: "actual-send", Status: "pending"}, nil
}
func (s *scheduledEmailTestSender) SendBulkEmails(context.Context, []*ComposedEmail, uint) ([]*SendResult, error) {
	return nil, nil
}
func (s *scheduledEmailTestSender) GetSendStatus(context.Context, string) (*SendStatus, error) {
	return nil, nil
}
func (s *scheduledEmailTestSender) ResendEmail(context.Context, string) (*SendResult, error) {
	return nil, nil
}
