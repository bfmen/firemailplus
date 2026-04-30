package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"firemail/internal/models"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSendTemplateQuotaSchemaMigratesAndSupportsCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "schema.db")), &gorm.Config{})
	require.NoError(t, err)

	createSchemaTestBaseTables(t, db)
	applyMigrationSQL(t, db,
		"000008_create_email_templates_table.up.sql",
		"000009_create_drafts_table.up.sql",
		"000015_create_send_queue_table.up.sql",
		"000021_fix_send_template_quota_schema.up.sql",
	)

	user := models.User{
		Username:    "schema-user",
		Password:    "password123",
		DisplayName: "Schema User",
		Role:        "admin",
		IsActive:    true,
	}
	require.NoError(t, db.Create(&user).Error)

	account := models.EmailAccount{
		UserID:       user.ID,
		Name:         "Schema Account",
		Email:        "schema@example.test",
		Provider:     "custom",
		AuthMethod:   "password",
		Username:     "schema@example.test",
		Password:     "app-password",
		IMAPHost:     "imap.example.test",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.example.test",
		SMTPPort:     587,
		SMTPSecurity: "STARTTLS",
		IsActive:     true,
	}
	require.NoError(t, db.Create(&account).Error)

	sent := models.SentEmail{
		SendID:     "send-schema-test",
		AccountID:  account.ID,
		MessageID:  "message-schema-test",
		Subject:    "schema subject",
		Recipients: "recipient@example.test",
		SentAt:     time.Now(),
		Status:     "sent",
	}
	require.NoError(t, db.Create(&sent).Error)

	template := models.EmailTemplate{
		Name:        "Schema Template",
		Description: "Template description",
		UserID:      user.ID,
		Subject:     "Hello {{name}}",
		TextBody:    "Hello {{name}}",
		HTMLBody:    "<p>Hello {{name}}</p>",
		Variables:   `[{"name":"name","type":"string","required":true}]`,
		IsActive:    true,
	}
	require.NoError(t, db.Create(&template).Error)

	quota := models.EmailQuota{
		UserID:              user.ID,
		DailyLimit:          10,
		MonthlyLimit:        100,
		AttachmentSizeLimit: 1024,
		LastResetDate:       time.Now(),
	}
	require.NoError(t, db.Create(&quota).Error)

	var draftTableCount int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'email_drafts'").Scan(&draftTableCount).Error)
	require.Zero(t, draftTableCount)
}

func createSchemaTestBaseTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.EmailAccount{}))
}

func applyMigrationSQL(t *testing.T, db *gorm.DB, names ...string) {
	t.Helper()

	for _, name := range names {
		file := filepath.Join("..", "..", "database", "migrations", name)
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		sql := stripSQLCommentLines(string(content))
		for _, statement := range strings.Split(sql, ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			require.NoError(t, db.Exec(statement).Error, "migration %s statement %q", file, statement)
		}
	}
}

func stripSQLCommentLines(sql string) string {
	var lines []string
	for _, line := range strings.Split(sql, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
