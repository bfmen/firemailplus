package services

import (
	"context"
	"mime"
	"strings"
	"testing"

	"firemail/internal/models"

	"github.com/stretchr/testify/require"
)

func TestStandardEmailComposerEncodesChineseHeadersAndAttachmentFilename(t *testing.T) {
	composer := NewStandardEmailComposer(nil, nil)

	email, err := composer.ComposeEmail(context.Background(), &ComposeEmailRequest{
		From:     &models.EmailAddress{Name: "发件人", Address: "sender@example.test"},
		To:       []*models.EmailAddress{{Name: "收件人", Address: "recipient@example.test"}},
		Subject:  "中文主题",
		TextBody: "plain body",
		Attachments: []*EmailAttachment{
			{
				Filename:    "报告.txt",
				ContentType: "text/plain",
				Data:        []byte("hello"),
				Size:        5,
			},
		},
	})
	require.NoError(t, err)

	raw := string(email.MIMEContent)
	require.Contains(t, raw, "Subject: "+mime.QEncoding.Encode("utf-8", "中文主题"))
	require.Contains(t, raw, mime.QEncoding.Encode("utf-8", "发件人")+" <sender@example.test>")
	require.Contains(t, raw, mime.QEncoding.Encode("utf-8", "收件人")+" <recipient@example.test>")
	require.Contains(t, raw, `filename="`+mime.QEncoding.Encode("utf-8", "报告.txt")+`"`)
	require.Contains(t, raw, "filename*=utf-8''%E6%8A%A5%E5%91%8A.txt")
	require.NotContains(t, strings.Split(raw, "\r\n\r\n")[0], "中文主题")
	require.NotContains(t, strings.Split(raw, "\r\n\r\n")[0], "报告.txt")
}
