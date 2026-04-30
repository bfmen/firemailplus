package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"firemail/internal/cache"
	"firemail/internal/models"
	"firemail/internal/providers"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSearchEmailsDoesNotMutateRequest(t *testing.T) {
	db := setupMaintainabilityTestDB(t)
	user, account := createMaintainabilityUserAccount(t, db)
	service := NewEmailService(db, providers.NewProviderFactory(), nil).(*EmailServiceImpl)
	service.cacheManager = cache.NewCacheManager()

	req := &SearchEmailsRequest{
		AccountID: &account.ID,
		Query:     `from:alice@example.test subject:"Quarterly Report" budget`,
		Page:      1,
		PageSize:  20,
	}
	before := *req

	_, err := service.SearchEmails(context.Background(), user.ID, req)
	require.NoError(t, err)
	require.Equal(t, before, *req)
}

func TestEmailListCacheInvalidationIsScopedToUser(t *testing.T) {
	cm := cache.NewCacheManager()
	service := &EmailServiceImpl{cacheManager: cm}

	userOneKey := service.generateEmailListCacheKey(1, &GetEmailsRequest{Page: 1})
	userTwoKey := service.generateEmailListCacheKey(2, &GetEmailsRequest{Page: 1})
	require.True(t, strings.HasPrefix(userOneKey, "emails:user:1:"))
	require.True(t, strings.HasPrefix(userTwoKey, "emails:user:2:"))

	cm.EmailListCache().Set(userOneKey, "user-one", time.Minute)
	cm.EmailListCache().Set(userTwoKey, "user-two", time.Minute)

	service.invalidateEmailListCache(1)

	_, foundOne := cm.EmailListCache().Get(userOneKey)
	userTwoValue, foundTwo := cm.EmailListCache().Get(userTwoKey)
	require.False(t, foundOne)
	require.True(t, foundTwo)
	require.Equal(t, "user-two", userTwoValue)
}

func TestReplySubjectForAddsSinglePrefix(t *testing.T) {
	require.Equal(t, "Re: Hello", replySubjectFor("Hello"))
	require.Equal(t, "Re: Hello", replySubjectFor("Re: Hello"))
	require.Equal(t, "re: Hello", replySubjectFor("re: Hello"))
	require.Equal(t, " Re: Hello", replySubjectFor(" Re: Hello"))
}

func TestComposerHTMLPolicyEscapesMarkup(t *testing.T) {
	composer := NewStandardEmailComposer(nil, nil)

	email, err := composer.ComposeEmail(context.Background(), &ComposeEmailRequest{
		From:     &models.EmailAddress{Name: "Sender", Address: "sender@example.test"},
		To:       []*models.EmailAddress{{Name: "Recipient", Address: "recipient@example.test"}},
		Subject:  "HTML policy",
		HTMLBody: `<p>safe</p><script>alert(1)</script>`,
	})
	require.NoError(t, err)
	require.Contains(t, email.HTMLBody, "&lt;p&gt;safe&lt;/p&gt;")
	require.Contains(t, email.HTMLBody, "&lt;script&gt;alert(1)&lt;/script&gt;")
	require.NotContains(t, email.HTMLBody, "<script>")
}

func setupMaintainabilityTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.EmailAccount{}, &models.Email{}))
	return db
}

func createMaintainabilityUserAccount(t *testing.T, db *gorm.DB) (*models.User, *models.EmailAccount) {
	t.Helper()

	user := &models.User{
		Username:    "maint-user",
		Password:    "password123",
		DisplayName: "Maintainability User",
		Role:        "user",
		IsActive:    true,
	}
	require.NoError(t, db.Create(user).Error)

	account := &models.EmailAccount{
		UserID:       user.ID,
		Name:         "Maintainability Account",
		Email:        "maint@example.test",
		Provider:     "custom",
		AuthMethod:   "password",
		Username:     "maint@example.test",
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
