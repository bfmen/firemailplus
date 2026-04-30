package services

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"
	"firemail/internal/providers"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStandardEmailSenderPersistsStatusAndResendCreatesNewSendID(t *testing.T) {
	db := setupEmailSenderTestDB(t)
	user, account := createEmailSenderTestAccount(t, db)
	smtp := &emailSenderTestSMTPClient{}
	factory := newEmailSenderTestFactory(smtp)
	sender := NewStandardEmailSender(db, factory, nil).(*StandardEmailSender)

	email := emailSenderTestComposedEmail("original")
	result, err := sender.SendEmail(context.Background(), email, account.ID)
	require.NoError(t, err)
	require.Equal(t, "pending", result.Status)

	require.Eventually(t, func() bool {
		var queue models.SendQueue
		err := db.Where("send_id = ?", result.SendID).First(&queue).Error
		return err == nil && queue.UserID == user.ID && queue.Status == "sent"
	}, time.Second, 10*time.Millisecond)

	restarted := NewStandardEmailSender(db, factory, nil).(*StandardEmailSender)
	status, err := restarted.GetSendStatus(context.Background(), result.SendID)
	require.NoError(t, err)
	require.Equal(t, "sent", status.Status)
	require.Equal(t, result.SendID, status.SendID)
	require.Equal(t, 1, status.SentRecipients)

	resend, err := restarted.ResendEmail(context.Background(), result.SendID)
	require.NoError(t, err)
	require.NotEqual(t, result.SendID, resend.SendID)
	require.Equal(t, "pending", resend.Status)

	require.Eventually(t, func() bool {
		var count int64
		err := db.Model(&models.SentEmail{}).Where("send_id IN ?", []string{result.SendID, resend.SendID}).Count(&count).Error
		return err == nil && count == 2
	}, time.Second, 10*time.Millisecond)
}

func TestStandardEmailSenderSendBulkEmailsRaceSafe(t *testing.T) {
	db := setupEmailSenderTestDB(t)
	_, account := createEmailSenderTestAccount(t, db)
	smtp := &emailSenderTestSMTPClient{}
	factory := newEmailSenderTestFactory(smtp)
	sender := NewStandardEmailSender(db, factory, nil).(*StandardEmailSender)

	emails := make([]*ComposedEmail, 25)
	for i := range emails {
		emails[i] = emailSenderTestComposedEmail(fmt.Sprintf("bulk-%d", i))
	}

	results, err := sender.SendBulkEmails(context.Background(), emails, account.ID)
	require.NoError(t, err)
	require.Len(t, results, len(emails))

	seen := make(map[string]bool, len(results))
	for _, result := range results {
		require.NotNil(t, result)
		require.NotEmpty(t, result.SendID)
		require.Equal(t, "pending", result.Status)
		require.False(t, seen[result.SendID], "duplicate send id %s", result.SendID)
		seen[result.SendID] = true
	}

	require.Eventually(t, func() bool {
		var count int64
		err := db.Model(&models.SendQueue{}).Where("status = ?", "sent").Count(&count).Error
		return err == nil && count == int64(len(emails))
	}, 2*time.Second, 10*time.Millisecond)
}

func setupEmailSenderTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.EmailAccount{},
		&models.SendQueue{},
		&models.SentEmail{},
	))
	return db
}

func createEmailSenderTestAccount(t *testing.T, db *gorm.DB) (*models.User, *models.EmailAccount) {
	t.Helper()

	user := &models.User{
		Username:    "sender-user",
		Password:    "password123",
		DisplayName: "Sender User",
		Role:        "user",
		IsActive:    true,
	}
	require.NoError(t, db.Create(user).Error)

	account := &models.EmailAccount{
		UserID:       user.ID,
		Name:         "Sender Account",
		Email:        "sender@example.test",
		Provider:     "custom",
		AuthMethod:   "password",
		Username:     "sender@example.test",
		IMAPHost:     "imap.example.test",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.example.test",
		SMTPPort:     587,
		SMTPSecurity: "STARTTLS",
		IsActive:     true,
	}
	require.NoError(t, db.Create(account).Error)
	return user, account
}

func newEmailSenderTestFactory(smtp *emailSenderTestSMTPClient) *providers.ProviderFactory {
	factory := providers.NewProviderFactory()
	factory.RegisterProvider("custom", func(*config.EmailProviderConfig) providers.EmailProvider {
		return &emailSenderTestProvider{smtp: smtp}
	})
	return factory
}

func emailSenderTestComposedEmail(id string) *ComposedEmail {
	return &ComposedEmail{
		ID:        id,
		From:      &models.EmailAddress{Name: "Sender", Address: "sender@example.test"},
		To:        []*models.EmailAddress{{Name: "Recipient", Address: "recipient@example.test"}},
		Subject:   "Subject " + id,
		TextBody:  "Body " + id,
		CreatedAt: time.Now(),
	}
}

type emailSenderTestProvider struct {
	smtp *emailSenderTestSMTPClient
}

func (p *emailSenderTestProvider) GetName() string                   { return "custom" }
func (p *emailSenderTestProvider) GetDisplayName() string            { return "Custom" }
func (p *emailSenderTestProvider) GetSupportedAuthMethods() []string { return []string{"password"} }
func (p *emailSenderTestProvider) GetProviderInfo() map[string]interface{} {
	return map[string]interface{}{}
}
func (p *emailSenderTestProvider) Connect(context.Context, *models.EmailAccount) error {
	return nil
}
func (p *emailSenderTestProvider) Disconnect() error                { return nil }
func (p *emailSenderTestProvider) IsConnected() bool                { return true }
func (p *emailSenderTestProvider) IsIMAPConnected() bool            { return false }
func (p *emailSenderTestProvider) IsSMTPConnected() bool            { return true }
func (p *emailSenderTestProvider) IMAPClient() providers.IMAPClient { return nil }
func (p *emailSenderTestProvider) SMTPClient() providers.SMTPClient { return p.smtp }
func (p *emailSenderTestProvider) OAuth2Client() providers.OAuth2Client {
	return nil
}
func (p *emailSenderTestProvider) TestConnection(context.Context, *models.EmailAccount) error {
	return nil
}
func (p *emailSenderTestProvider) SendEmail(context.Context, *models.EmailAccount, *providers.OutgoingMessage) error {
	return nil
}
func (p *emailSenderTestProvider) SyncEmails(context.Context, *models.EmailAccount, string, uint32) ([]*providers.EmailMessage, error) {
	return nil, nil
}

type emailSenderTestSMTPClient struct {
	mu       sync.Mutex
	messages []*providers.OutgoingMessage
}

func (c *emailSenderTestSMTPClient) Connect(context.Context, providers.SMTPClientConfig) error {
	return nil
}
func (c *emailSenderTestSMTPClient) Disconnect() error { return nil }
func (c *emailSenderTestSMTPClient) IsConnected() bool { return true }
func (c *emailSenderTestSMTPClient) SendRawEmail(context.Context, string, []string, []byte) error {
	return nil
}
func (c *emailSenderTestSMTPClient) SendEmail(_ context.Context, message *providers.OutgoingMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, message)
	return nil
}
