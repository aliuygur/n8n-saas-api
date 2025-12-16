package gcplog

import (
	"log/slog"
	"net/http"
	"time"
)

// HTTPRequest represents the GCP Cloud Logging httpRequest structure
// See: https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#HttpRequest
type HTTPRequest struct {
	RequestMethod string `json:"requestMethod,omitempty"`
	RequestURL    string `json:"requestUrl,omitempty"`
	RequestSize   int64  `json:"requestSize,omitempty,string"`
	Status        int    `json:"status,omitempty"`
	ResponseSize  int64  `json:"responseSize,omitempty,string"`
	UserAgent     string `json:"userAgent,omitempty"`
	RemoteIP      string `json:"remoteIp,omitempty"`
	ServerIP      string `json:"serverIp,omitempty"`
	Referer       string `json:"referer,omitempty"`
	Latency       string `json:"latency,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	status      int
	bytesWritten int64
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

// HTTPMiddleware returns a middleware that logs HTTP requests in GCP format
func HTTPMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status and size
			wrapped := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK, // default
			}

			// Call next handler
			next.ServeHTTP(wrapped, r)

			// Calculate latency
			latency := time.Since(start)

			// Build httpRequest object for GCP
			httpReq := HTTPRequest{
				RequestMethod: r.Method,
				RequestURL:    r.URL.String(),
				Status:        wrapped.status,
				ResponseSize:  wrapped.bytesWritten,
				UserAgent:     r.UserAgent(),
				RemoteIP:      r.RemoteAddr,
				Referer:       r.Referer(),
				Latency:       formatDuration(latency),
				Protocol:      r.Proto,
			}

			// Determine log level based on status code
			level := slog.LevelInfo
			if wrapped.status >= 500 {
				level = slog.LevelError
			} else if wrapped.status >= 400 {
				level = slog.LevelWarn
			}

			// Log with GCP httpRequest field
			logger.Log(r.Context(), level, "HTTP request",
				"httpRequest", httpReq,
				"path", r.URL.Path,
				"method", r.Method,
				"status", wrapped.status,
				"duration_ms", latency.Milliseconds(),
			)
		})
	}
}

// formatDuration formats a duration as a string with seconds and fractional seconds
// GCP expects format like "1.234s"
func formatDuration(d time.Duration) string {
	return d.String()
}
