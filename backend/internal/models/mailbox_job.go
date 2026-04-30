package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const (
	MailboxJobOperationMarkAccountsRead = "mark_accounts_read"

	MailboxJobStatusQueued    = "queued"
	MailboxJobStatusRunning   = "running"
	MailboxJobStatusCompleted = "completed"
	MailboxJobStatusFailed    = "failed"
)

// MailboxJob stores durable state for mailbox actions that must not block the
// HTTP request path.
type MailboxJob struct {
	BaseModel

	JobID     string `gorm:"uniqueIndex;size:100;not null" json:"job_id"`
	UserID    uint   `gorm:"index;not null" json:"user_id"`
	Operation string `gorm:"size:50;not null;index" json:"operation"`
	Status    string `gorm:"size:50;not null;default:'queued';index" json:"status"`

	AccountIDs     []uint `gorm:"-" json:"account_ids"`
	AccountIDsJSON string `gorm:"column:account_ids;type:text;not null" json:"-"`

	ProcessedCount int `gorm:"not null;default:0" json:"processed_count"`
	TotalCount     int `gorm:"not null;default:0" json:"total_count"`

	ErrorMessage string     `gorm:"column:error_message;type:text" json:"error,omitempty"`
	StartedAt    *time.Time `gorm:"index" json:"started_at,omitempty"`
	CompletedAt  *time.Time `gorm:"index" json:"completed_at,omitempty"`
}

func (MailboxJob) TableName() string {
	return "mailbox_jobs"
}

func (j *MailboxJob) SetAccountIDs(accountIDs []uint) error {
	j.AccountIDs = append([]uint(nil), accountIDs...)
	data, err := json.Marshal(j.AccountIDs)
	if err != nil {
		return err
	}
	j.AccountIDsJSON = string(data)
	return nil
}

func (j *MailboxJob) GetAccountIDs() ([]uint, error) {
	if len(j.AccountIDs) > 0 {
		return append([]uint(nil), j.AccountIDs...), nil
	}
	if j.AccountIDsJSON == "" {
		return []uint{}, nil
	}

	var accountIDs []uint
	if err := json.Unmarshal([]byte(j.AccountIDsJSON), &accountIDs); err != nil {
		return nil, err
	}
	j.AccountIDs = append([]uint(nil), accountIDs...)
	return append([]uint(nil), accountIDs...), nil
}

func (j *MailboxJob) BeforeSave(_ *gorm.DB) error {
	if j.AccountIDs == nil {
		return nil
	}
	return j.SetAccountIDs(j.AccountIDs)
}

func (j *MailboxJob) AfterFind(_ *gorm.DB) error {
	_, err := j.GetAccountIDs()
	return err
}
