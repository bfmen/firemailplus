package services

import (
	"context"
	"strings"
	"testing"

	"firemail/internal/models"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStandardEmailComposerProcessesInjectedTemplate(t *testing.T) {
	db := setupTemplateComposerTestDB(t)
	user := createTemplateComposerUser(t, db)
	templateService := NewEmailTemplateService(db)

	tmpl, err := templateService.CreateTemplate(context.Background(), user.ID, &CreateEmailTemplateRequest{
		Name:     "Greeting",
		Subject:  "Hello {{.name}}",
		TextBody: "Plain {{.name}}",
		HTMLBody: "<p>HTML {{.name}}</p>",
	})
	require.NoError(t, err)

	composer := NewStandardEmailComposer(nil, db).(*StandardEmailComposer)
	composer.SetTemplateService(templateService)

	email, err := composer.ComposeEmail(context.Background(), &ComposeEmailRequest{
		From:         &models.EmailAddress{Name: "Sender", Address: "sender@example.test"},
		To:           []*models.EmailAddress{{Name: "Recipient", Address: "recipient@example.test"}},
		TemplateID:   &tmpl.ID,
		TemplateData: map[string]interface{}{"name": "Ada"},
	})
	require.NoError(t, err)
	require.Equal(t, "Hello Ada", email.Subject)
	require.Equal(t, "Plain Ada", email.TextBody)
	require.Equal(t, "&lt;p&gt;HTML Ada&lt;/p&gt;", email.HTMLBody)
	require.NotEmpty(t, email.MIMEContent)
}

func TestStandardEmailComposerFailsWhenTemplateVariableMissing(t *testing.T) {
	db := setupTemplateComposerTestDB(t)
	user := createTemplateComposerUser(t, db)
	templateService := NewEmailTemplateService(db)

	tmpl, err := templateService.CreateTemplate(context.Background(), user.ID, &CreateEmailTemplateRequest{
		Name:     "Needs Name",
		Subject:  "Hello {{.name}}",
		TextBody: "Plain {{.name}}",
	})
	require.NoError(t, err)

	composer := NewStandardEmailComposer(nil, db).(*StandardEmailComposer)
	composer.SetTemplateService(templateService)

	_, err = composer.ComposeEmail(context.Background(), &ComposeEmailRequest{
		From:       &models.EmailAddress{Name: "Sender", Address: "sender@example.test"},
		To:         []*models.EmailAddress{{Name: "Recipient", Address: "recipient@example.test"}},
		TemplateID: &tmpl.ID,
	})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "map has no entry") || strings.Contains(err.Error(), "failed to process"))
}

func setupTemplateComposerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.EmailTemplate{}))
	return db
}

func createTemplateComposerUser(t *testing.T, db *gorm.DB) *models.User {
	t.Helper()

	user := &models.User{
		Username:    "template-user",
		Password:    "password123",
		DisplayName: "Template User",
		Role:        "user",
		IsActive:    true,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}
