package middleware

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"firemail/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type rejectingAuthService struct{}

func (rejectingAuthService) ValidateToken(string) (*models.User, error) {
	return nil, errors.New("invalid token")
}

func TestAuthRequiredDoesNotLogTokenMaterial(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logs bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(originalOutput) })

	router := gin.New()
	router.GET("/private", AuthRequiredWithService(rejectingAuthService{}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer secret-token-value-that-must-not-appear")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
	require.NotContains(t, logs.String(), "secret-token-value-that-must-not-appear")
	require.NotContains(t, logs.String(), "Bearer secret-token")
	require.Contains(t, logs.String(), "Authorization header present: true")
}
