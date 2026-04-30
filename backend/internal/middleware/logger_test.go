package middleware

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactSensitiveQuery(t *testing.T) {
	redacted := RedactSensitiveQuery("/api/v1/sse?token=secret-token&client_id=abc")

	require.NotContains(t, redacted, "secret-token")
	require.Contains(t, redacted, "token=%5BREDACTED%5D")
	require.Contains(t, redacted, "client_id=abc")
}

func TestRedactSensitiveQueryLeavesSafePath(t *testing.T) {
	require.Equal(t, "/api/v1/emails?page=1", RedactSensitiveQuery("/api/v1/emails?page=1"))
}
