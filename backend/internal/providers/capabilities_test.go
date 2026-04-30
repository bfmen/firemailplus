package providers

import (
	"context"
	"fmt"
	"io"
	"testing"

	"firemail/internal/models"

	"github.com/stretchr/testify/require"
)

func TestCapabilityDetectorSMTPRequiresRealConnectionCheck(t *testing.T) {
	smtp := &capabilityTestSMTPClient{}
	detector := NewCapabilityDetector(&capabilityTestProvider{smtp: smtp})

	supported, err := detector.TestFeature(context.Background(), &models.EmailAccount{
		AuthMethod: "password",
		Username:   "user@example.test",
		Password:   "secret",
	}, "smtp")
	require.Error(t, err)
	require.False(t, supported)
	require.False(t, smtp.connectCalled)

	supported, err = detector.TestFeature(context.Background(), &models.EmailAccount{
		AuthMethod:   "password",
		Username:     "user@example.test",
		Password:     "secret",
		SMTPHost:     "smtp.example.test",
		SMTPPort:     587,
		SMTPSecurity: "STARTTLS",
	}, "smtp")
	require.NoError(t, err)
	require.True(t, supported)
	require.True(t, smtp.connectCalled)
	require.Equal(t, "smtp.example.test", smtp.config.Host)
	require.Equal(t, 587, smtp.config.Port)
	require.Equal(t, "STARTTLS", smtp.config.Security)
}

func TestCapabilityDetectorSMTPFailureIsNotReportedAsSupported(t *testing.T) {
	smtp := &capabilityTestSMTPClient{connectErr: fmt.Errorf("dial failed")}
	detector := NewCapabilityDetector(&capabilityTestProvider{smtp: smtp})

	supported, err := detector.TestFeature(context.Background(), &models.EmailAccount{
		AuthMethod:   "password",
		Username:     "user@example.test",
		Password:     "secret",
		SMTPHost:     "smtp.example.test",
		SMTPPort:     587,
		SMTPSecurity: "STARTTLS",
	}, "smtp")
	require.Error(t, err)
	require.False(t, supported)
}

type capabilityTestProvider struct {
	smtp SMTPClient
}

func (p *capabilityTestProvider) GetName() string                   { return "test" }
func (p *capabilityTestProvider) GetDisplayName() string            { return "Test" }
func (p *capabilityTestProvider) GetSupportedAuthMethods() []string { return []string{"password"} }
func (p *capabilityTestProvider) GetProviderInfo() map[string]interface{} {
	return map[string]interface{}{}
}
func (p *capabilityTestProvider) Connect(context.Context, *models.EmailAccount) error { return nil }
func (p *capabilityTestProvider) Disconnect() error                                   { return nil }
func (p *capabilityTestProvider) IsConnected() bool                                   { return true }
func (p *capabilityTestProvider) IsIMAPConnected() bool                               { return false }
func (p *capabilityTestProvider) IsSMTPConnected() bool                               { return p.smtp != nil && p.smtp.IsConnected() }
func (p *capabilityTestProvider) TestConnection(context.Context, *models.EmailAccount) error {
	return nil
}
func (p *capabilityTestProvider) IMAPClient() IMAPClient { return nil }
func (p *capabilityTestProvider) SMTPClient() SMTPClient { return p.smtp }
func (p *capabilityTestProvider) OAuth2Client() OAuth2Client {
	return nil
}
func (p *capabilityTestProvider) SendEmail(context.Context, *models.EmailAccount, *OutgoingMessage) error {
	return nil
}
func (p *capabilityTestProvider) SyncEmails(context.Context, *models.EmailAccount, string, uint32) ([]*EmailMessage, error) {
	return nil, nil
}

type capabilityTestSMTPClient struct {
	connectCalled bool
	connected     bool
	config        SMTPClientConfig
	connectErr    error
}

func (c *capabilityTestSMTPClient) Connect(_ context.Context, config SMTPClientConfig) error {
	c.connectCalled = true
	c.config = config
	if c.connectErr != nil {
		return c.connectErr
	}
	c.connected = true
	return nil
}
func (c *capabilityTestSMTPClient) Disconnect() error {
	c.connected = false
	return nil
}
func (c *capabilityTestSMTPClient) IsConnected() bool { return c.connected }
func (c *capabilityTestSMTPClient) SendRawEmail(context.Context, string, []string, []byte) error {
	return nil
}
func (c *capabilityTestSMTPClient) SendEmail(context.Context, *OutgoingMessage) error {
	return nil
}

type capabilityTestIMAPClient struct{}

func (c *capabilityTestIMAPClient) Connect(context.Context, IMAPClientConfig) error { return nil }
func (c *capabilityTestIMAPClient) Disconnect() error                               { return nil }
func (c *capabilityTestIMAPClient) IsConnected() bool                               { return true }
func (c *capabilityTestIMAPClient) ListFolders(context.Context) ([]*FolderInfo, error) {
	return nil, nil
}
func (c *capabilityTestIMAPClient) SelectFolder(context.Context, string) (*FolderStatus, error) {
	return nil, nil
}
func (c *capabilityTestIMAPClient) CreateFolder(context.Context, string) error { return nil }
func (c *capabilityTestIMAPClient) DeleteFolder(context.Context, string) error { return nil }
func (c *capabilityTestIMAPClient) RenameFolder(context.Context, string, string) error {
	return nil
}
func (c *capabilityTestIMAPClient) FetchEmails(context.Context, *FetchCriteria) ([]*EmailMessage, error) {
	return nil, nil
}
func (c *capabilityTestIMAPClient) FetchEmailByUID(context.Context, uint32) (*EmailMessage, error) {
	return nil, nil
}
func (c *capabilityTestIMAPClient) FetchEmailHeaders(context.Context, []uint32) ([]*EmailHeader, error) {
	return nil, nil
}
func (c *capabilityTestIMAPClient) MarkAsRead(context.Context, []uint32) error   { return nil }
func (c *capabilityTestIMAPClient) MarkAsUnread(context.Context, []uint32) error { return nil }
func (c *capabilityTestIMAPClient) DeleteEmails(context.Context, []uint32) error { return nil }
func (c *capabilityTestIMAPClient) MoveEmails(context.Context, []uint32, string) error {
	return nil
}
func (c *capabilityTestIMAPClient) CopyEmails(context.Context, []uint32, string) error {
	return nil
}
func (c *capabilityTestIMAPClient) SearchEmails(context.Context, *SearchCriteria) ([]uint32, error) {
	return nil, nil
}
func (c *capabilityTestIMAPClient) GetFolderStatus(context.Context, string) (*FolderStatus, error) {
	return nil, nil
}
func (c *capabilityTestIMAPClient) GetNewEmails(context.Context, string, uint32) ([]*EmailMessage, error) {
	return nil, nil
}
func (c *capabilityTestIMAPClient) GetEmailsInUIDRange(context.Context, string, uint32, uint32) ([]*EmailMessage, error) {
	return nil, nil
}
func (c *capabilityTestIMAPClient) GetAttachment(context.Context, string, uint32, string) (io.ReadCloser, error) {
	return nil, nil
}
