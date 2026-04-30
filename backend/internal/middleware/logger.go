package middleware

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

// RedactedLogger is a Gin access logger that redacts query parameters carrying
// credentials, including the SSE EventSource compatibility token.
func RedactedLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[GIN] %v |%3d| %13v | %15s |%-7s %#v\n%s",
			param.TimeStamp.Format(time.RFC3339),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			RedactSensitiveQuery(param.Path),
			param.ErrorMessage,
		)
	})
}

func RedactSensitiveQuery(rawPath string) string {
	parsed, err := url.ParseRequestURI(rawPath)
	if err != nil {
		return rawPath
	}

	query := parsed.Query()
	changed := false
	for _, key := range []string{"token", "access_token", "refresh_token"} {
		if _, ok := query[key]; ok {
			query.Set(key, "[REDACTED]")
			changed = true
		}
	}
	if !changed {
		return rawPath
	}

	parsed.RawQuery = query.Encode()
	return parsed.RequestURI()
}
