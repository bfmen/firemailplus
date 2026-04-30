package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"firemail/internal/models"
	"firemail/internal/providers"
	"firemail/internal/sse"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type safeMailboxJobPublisher struct {
	mu     sync.Mutex
	events []*sse.Event
}

func (p *safeMailboxJobPublisher) Publish(_ context.Context, event *sse.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if event != nil {
		p.events = append(p.events, event)
	}
	return nil
}

func (p *safeMailboxJobPublisher) PublishToUser(ctx context.Context, userID uint, event *sse.Event) error {
	if event != nil {
		event.UserID = userID
	}
	return p.Publish(ctx, event)
}

func (p *safeMailboxJobPublisher) PublishToAccount(ctx context.Context, _ uint, event *sse.Event) error {
	return p.Publish(ctx, event)
}

func (p *safeMailboxJobPublisher) Broadcast(ctx context.Context, event *sse.Event) error {
	return p.Publish(ctx, event)
}

func (p *safeMailboxJobPublisher) snapshot() []*sse.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]*sse.Event(nil), p.events...)
}

type mailboxJobTestEnv struct {
	db        *gorm.DB
	service   *EmailServiceImpl
	publisher *safeMailboxJobPublisher
	user      *models.User
	otherUser *models.User
	accountA  *models.EmailAccount
	accountB  *models.EmailAccount
}

func setupMailboxJobTestEnv(t *testing.T) *mailboxJobTestEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.EmailAccount{},
		&models.MailboxJob{},
	))

	user := createMailboxJobUser(t, db, "mailbox_job_user")
	otherUser := createMailboxJobUser(t, db, "mailbox_job_other")
	accountA := createMailboxJobAccount(t, db, user.ID, "account-a@example.test")
	accountB := createMailboxJobAccount(t, db, user.ID, "account-b@example.test")

	publisher := &safeMailboxJobPublisher{}
	service, ok := NewEmailService(db, providers.NewProviderFactory(), publisher).(*EmailServiceImpl)
	require.True(t, ok)

	return &mailboxJobTestEnv{
		db:        db,
		service:   service,
		publisher: publisher,
		user:      user,
		otherUser: otherUser,
		accountA:  accountA,
		accountB:  accountB,
	}
}

func createMailboxJobUser(t *testing.T, db *gorm.DB, username string) *models.User {
	t.Helper()

	user := &models.User{
		Username: username + "_" + fmt.Sprint(time.Now().UnixNano()),
		Password: "password123",
		Role:     "user",
		IsActive: true,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createMailboxJobAccount(t *testing.T, db *gorm.DB, userID uint, email string) *models.EmailAccount {
	t.Helper()

	account := &models.EmailAccount{
		UserID:       userID,
		Name:         email,
		Email:        email,
		Provider:     "custom",
		AuthMethod:   "password",
		IMAPHost:     "imap.example.test",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		IsActive:     true,
		SyncStatus:   "pending",
	}
	require.NoError(t, db.Create(account).Error)
	return account
}

func waitForMailboxJobStatus(t *testing.T, service *EmailServiceImpl, userID uint, jobID, status string) *models.MailboxJob {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, err := service.GetMailboxJob(context.Background(), userID, jobID)
		if err == nil && job.Status == status {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}

	job, err := service.GetMailboxJob(context.Background(), userID, jobID)
	require.NoError(t, err)
	require.Equal(t, status, job.Status)
	return job
}

func TestStartMarkAccountsAsReadJobReturnsBeforeWorkerCompletes(t *testing.T) {
	env := setupMailboxJobTestEnv(t)
	ctx := context.Background()

	started := make(chan uint, 2)
	release := make(chan struct{})
	env.service.markAccountAsReadJobRunner = func(ctx context.Context, userID, accountID uint) error {
		started <- accountID
		<-release
		return nil
	}

	startedAt := time.Now()
	job, err := env.service.StartMarkAccountsAsReadJob(ctx, env.user.ID, []uint{env.accountA.ID, env.accountB.ID})
	require.NoError(t, err)
	require.Less(t, time.Since(startedAt), 250*time.Millisecond)
	require.Equal(t, models.MailboxJobStatusQueued, job.Status)
	require.Equal(t, 2, job.TotalCount)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("mark-read worker did not start")
	}

	running := waitForMailboxJobStatus(t, env.service, env.user.ID, job.JobID, models.MailboxJobStatusRunning)
	require.Equal(t, 0, running.ProcessedCount)

	close(release)

	completed := waitForMailboxJobStatus(t, env.service, env.user.ID, job.JobID, models.MailboxJobStatusCompleted)
	require.Equal(t, 2, completed.ProcessedCount)
	require.Equal(t, []uint{env.accountA.ID, env.accountB.ID}, completed.AccountIDs)
	require.NotNil(t, completed.StartedAt)
	require.NotNil(t, completed.CompletedAt)

	events := env.publisher.snapshot()
	require.NotEmpty(t, events)
	var sawCompleted bool
	for _, event := range events {
		if event.Type != sse.EventMailboxJobUpdated {
			continue
		}
		data, ok := event.Data.(*sse.MailboxJobEventData)
		require.True(t, ok)
		if data.JobID == job.JobID && data.Status == models.MailboxJobStatusCompleted {
			sawCompleted = true
		}
	}
	require.True(t, sawCompleted, "expected completed mailbox job SSE event")
}

func TestStartMarkAccountsAsReadJobRecordsFailure(t *testing.T) {
	env := setupMailboxJobTestEnv(t)
	ctx := context.Background()
	expectedErr := errors.New("remote IMAP writeback failed")

	env.service.markAccountAsReadJobRunner = func(_ context.Context, _ uint, accountID uint) error {
		if accountID == env.accountB.ID {
			return expectedErr
		}
		return nil
	}

	job, err := env.service.StartMarkAccountsAsReadJob(ctx, env.user.ID, []uint{env.accountA.ID, env.accountB.ID})
	require.NoError(t, err)

	failed := waitForMailboxJobStatus(t, env.service, env.user.ID, job.JobID, models.MailboxJobStatusFailed)
	require.Equal(t, 1, failed.ProcessedCount)
	require.Equal(t, expectedErr.Error(), failed.ErrorMessage)
	require.NotNil(t, failed.CompletedAt)
}

func TestMailboxJobAccessIsUserScoped(t *testing.T) {
	env := setupMailboxJobTestEnv(t)
	ctx := context.Background()

	env.service.markAccountAsReadJobRunner = func(context.Context, uint, uint) error { return nil }
	job, err := env.service.StartMarkAccountsAsReadJob(ctx, env.user.ID, []uint{env.accountA.ID})
	require.NoError(t, err)

	_, err = env.service.GetMailboxJob(ctx, env.otherUser.ID, job.JobID)
	require.Error(t, err)
	require.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	otherAccount := createMailboxJobAccount(t, env.db, env.otherUser.ID, "other@example.test")
	_, err = env.service.StartMarkAccountsAsReadJob(ctx, env.user.ID, []uint{otherAccount.ID})
	require.Error(t, err)
}
