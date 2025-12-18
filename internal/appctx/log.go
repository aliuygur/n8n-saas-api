package appctx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"

	"github.com/samber/lo"
)

type loggerContextKey struct{}

// LoggerMiddleware returns a middleware that injects the provided logger into the request context
func LoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate or extract a request ID (here, use X-Request-Id header or generate a new one)
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = generateRequestID()
			}

			attrs := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			}
			reqLogger := logger.With(lo.ToAnySlice(attrs)...)
			ctx := context.WithValue(r.Context(), loggerContextKey{}, reqLogger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetLogger retrieves the logger from the context. Returns nil if not found.
func GetLogger(ctx context.Context) *slog.Logger {
	logger, _ := ctx.Value(loggerContextKey{}).(*slog.Logger)
	return logger
}

// generateRequestID generates a simple unique request ID (16 random hex chars)
func generateRequestID() string {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}
