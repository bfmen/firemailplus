package handlers

import (
	"errors"
	"net/http"
	"testing"

	"firemail/internal/services"

	"github.com/stretchr/testify/require"
)

func TestEmailReadStateHTTPErrorMapsTypedRemoteFailure(t *testing.T) {
	err := &services.EmailReadStateError{
		Code:      services.EmailReadStateRemoteSyncFailedCode,
		Operation: services.EmailReadStateOperationMarkRead,
		Err:       errors.New("failed to select folder: archive unavailable"),
	}

	status, code, message := emailReadStateHTTPError(services.EmailReadStateOperationMarkRead, err)

	require.Equal(t, http.StatusBadGateway, status)
	require.Equal(t, services.EmailReadStateRemoteSyncFailedCode, code)
	require.Contains(t, message, "Remote sync failed")
	require.Contains(t, message, "archive unavailable")
}

func TestEmailReadStateHTTPErrorMapsTypedNotSyncableFailure(t *testing.T) {
	err := &services.EmailReadStateError{
		Code:      services.EmailReadStateNotSyncableCode,
		Operation: services.EmailReadStateOperationMarkRead,
		Err:       errors.New("email cannot sync read state to server: missing UID"),
	}

	status, code, message := emailReadStateHTTPError(services.EmailReadStateOperationMarkRead, err)

	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, services.EmailReadStateNotSyncableCode, code)
	require.Contains(t, message, "Cannot mark email as read")
	require.Contains(t, message, "missing UID")
}

func TestEmailReadStateHTTPErrorMapsNotFound(t *testing.T) {
	status, code, message := emailReadStateHTTPError(services.EmailReadStateOperationMarkUnread, errors.New("email not found"))

	require.Equal(t, http.StatusNotFound, status)
	require.Empty(t, code)
	require.Equal(t, "Email not found", message)
}
