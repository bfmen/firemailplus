package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"firemail/internal/models"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAttachmentPreviewUnsupportedReturnsStableCode(t *testing.T) {
	db := setupAttachmentPreviewTestDB(t)
	userID := uint(77)
	attachment := &models.Attachment{
		UserID:      &userID,
		Filename:    "archive.zip",
		ContentType: "application/zip",
		Size:        4,
	}
	require.NoError(t, db.Create(attachment).Error)

	storage := NewLocalFileStorage(&AttachmentStorageConfig{
		BaseDir:      t.TempDir(),
		MaxFileSize:  1024,
		CompressText: false,
		CreateDirs:   true,
		ChecksumType: "md5",
	})
	service := NewAttachmentService(db, storage, nil).(*AttachmentService)

	preview, err := service.PreviewAttachment(context.Background(), attachment.ID, userID)
	require.Error(t, err)
	require.True(t, IsAttachmentPreviewUnsupported(err))
	require.Equal(t, AttachmentPreviewUnsupportedCode, preview.Code)
	require.Equal(t, "unknown", preview.Type)
	require.NotEmpty(t, preview.Error)
	require.Empty(t, preview.Text)
	require.Empty(t, preview.Content)
}

func TestAttachmentPDFPreviewDoesNotPretendImplemented(t *testing.T) {
	db := setupAttachmentPreviewTestDB(t)
	userID := uint(78)
	attachment := &models.Attachment{
		UserID:      &userID,
		Filename:    "file.pdf",
		ContentType: "application/pdf",
		Size:        8,
	}
	require.NoError(t, db.Create(attachment).Error)

	storage := NewLocalFileStorage(&AttachmentStorageConfig{
		BaseDir:      t.TempDir(),
		MaxFileSize:  1024,
		CompressText: false,
		CreateDirs:   true,
		ChecksumType: "md5",
	})
	require.NoError(t, storage.Store(context.Background(), attachment, strings.NewReader("%PDF-1.7")))
	require.NoError(t, db.Save(attachment).Error)

	service := NewAttachmentService(db, storage, nil).(*AttachmentService)
	preview, err := service.PreviewAttachment(context.Background(), attachment.ID, userID)
	require.Error(t, err)
	require.True(t, IsAttachmentPreviewUnsupported(err))
	require.Equal(t, AttachmentPreviewUnsupportedCode, preview.Code)
	require.Equal(t, "pdf", preview.Type)
	require.NotContains(t, preview.Text, "not implemented")
}

func TestLocalFileStorageChecksumConfigIsTruthful(t *testing.T) {
	storage := NewLocalFileStorage(&AttachmentStorageConfig{
		BaseDir:      t.TempDir(),
		MaxFileSize:  1024,
		CompressText: false,
		CreateDirs:   true,
		ChecksumType: "sha256",
	})
	attachment := &models.Attachment{Filename: "note.txt", Size: 5}

	require.NoError(t, storage.Store(context.Background(), attachment, strings.NewReader("hello")))
	info, err := storage.GetStorageInfo(context.Background(), attachment)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256([]byte("hello"))), info.Checksum)
	require.False(t, info.IsCompressed)
}

func TestLocalFileStorageRejectsUnsupportedChecksumType(t *testing.T) {
	storage := NewLocalFileStorage(&AttachmentStorageConfig{
		BaseDir:      t.TempDir(),
		MaxFileSize:  1024,
		CompressText: false,
		CreateDirs:   true,
		ChecksumType: "crc32",
	})
	attachment := &models.Attachment{Filename: "note.txt", Size: 5}

	err := storage.Store(context.Background(), attachment, strings.NewReader("hello"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported attachment checksum type")
}

func setupAttachmentPreviewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Attachment{}))
	return db
}
