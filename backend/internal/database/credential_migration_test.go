package database

import (
	"fmt"
	"testing"

	"firemail/internal/security"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateEmailAccountCredentialsToEncrypted(t *testing.T) {
	_, err := security.ConfigureFieldEncryption(security.EncryptionConfig{JWTSecret: "migration-test-jwt-secret"})
	require.NoError(t, err)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE email_accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			password TEXT,
			oauth2_token TEXT
		)
	`).Error)

	alreadyEncrypted, err := security.EncryptString("already-secret")
	require.NoError(t, err)

	require.NoError(t, db.Exec(
		"INSERT INTO email_accounts (password, oauth2_token) VALUES (?, ?), (?, ?), (?, ?)",
		"legacy-password",
		`{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expiry":"2026-01-01T00:00:00Z"}`,
		alreadyEncrypted,
		"",
		"",
		"",
	).Error)

	require.NoError(t, migrateEmailAccountCredentialsToEncrypted(db))

	var rows []emailAccountCredentialRow
	require.NoError(t, db.Raw("SELECT id, password, oauth2_token FROM email_accounts ORDER BY id").Scan(&rows).Error)
	require.Len(t, rows, 3)

	require.True(t, security.IsEncryptedString(rows[0].Password))
	require.True(t, security.IsEncryptedString(rows[0].OAuth2Token))
	require.Equal(t, alreadyEncrypted, rows[1].Password)
	require.Empty(t, rows[1].OAuth2Token)
	require.Empty(t, rows[2].Password)
	require.Empty(t, rows[2].OAuth2Token)

	password, legacy, err := security.DecryptString(rows[0].Password)
	require.NoError(t, err)
	require.False(t, legacy)
	require.Equal(t, "legacy-password", password)
}
