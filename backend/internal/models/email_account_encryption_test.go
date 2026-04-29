package models

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"firemail/internal/security"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupEmailAccountEncryptionDB(t *testing.T) *gorm.DB {
	t.Helper()

	_, err := security.ConfigureFieldEncryption(security.EncryptionConfig{JWTSecret: "model-test-jwt-secret"})
	require.NoError(t, err)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &EmailGroup{}, &EmailAccount{}))
	return db
}

func TestEmailAccountEncryptsCredentialsAtRestAndDecryptsAfterFind(t *testing.T) {
	db := setupEmailAccountEncryptionDB(t)

	user := &User{Username: "user", Password: "password123", Role: "admin", IsActive: true}
	require.NoError(t, db.Create(user).Error)

	token := &OAuth2TokenData{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
		ClientID:     "client-id",
	}

	account := &EmailAccount{
		UserID:     user.ID,
		Name:       "Account",
		Email:      "user@example.com",
		Provider:   "gmail",
		AuthMethod: "oauth2",
		Username:   "user@example.com",
		Password:   "app-password",
		IsActive:   true,
		SyncStatus: "pending",
	}
	require.NoError(t, account.SetOAuth2Token(token))
	require.NoError(t, db.Create(account).Error)

	var raw struct {
		Password    string
		OAuth2Token string `gorm:"column:oauth2_token"`
	}
	require.NoError(t, db.Raw("SELECT password, oauth2_token FROM email_accounts WHERE id = ?", account.ID).Scan(&raw).Error)
	require.True(t, security.IsEncryptedString(raw.Password))
	require.True(t, security.IsEncryptedString(raw.OAuth2Token))
	require.NotContains(t, raw.Password, "app-password")
	require.NotContains(t, raw.OAuth2Token, "refresh-token")

	var loaded EmailAccount
	require.NoError(t, db.First(&loaded, account.ID).Error)
	require.Equal(t, "app-password", loaded.Password)

	loadedToken, err := loaded.GetOAuth2Token()
	require.NoError(t, err)
	require.Equal(t, token.AccessToken, loadedToken.AccessToken)
	require.Equal(t, token.RefreshToken, loadedToken.RefreshToken)
}

func TestEmailAccountReadsLegacyPlainOAuth2Token(t *testing.T) {
	_, err := security.ConfigureFieldEncryption(security.EncryptionConfig{JWTSecret: "model-test-jwt-secret"})
	require.NoError(t, err)

	token := &OAuth2TokenData{
		AccessToken:  "legacy-access-token",
		RefreshToken: "legacy-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	tokenBytes, err := json.Marshal(token)
	require.NoError(t, err)

	account := &EmailAccount{OAuth2Token: string(tokenBytes)}
	loadedToken, err := account.GetOAuth2Token()
	require.NoError(t, err)
	require.Equal(t, token.AccessToken, loadedToken.AccessToken)
	require.Equal(t, token.RefreshToken, loadedToken.RefreshToken)
}
