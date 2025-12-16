# gcplog - GCP Cloud Logging Handler for slog

A custom `slog.Handler` implementation optimized for Google Cloud Platform (GCP) Cloud Logging. This package provides structured logging that integrates seamlessly with GCP's logging infrastructure.

## Features

- **GCP-Compatible Format**: Outputs logs in JSON format with GCP-specific fields
- **Severity Mapping**: Automatically maps slog levels to GCP severity levels (DEBUG, INFO, WARNING, ERROR)
- **Source Location**: Includes file, line, and function information for better debugging
- **HTTP Request Logging**: Built-in middleware for logging HTTP requests with GCP's httpRequest format
- **Trace Support**: Native support for distributed tracing with Cloud Trace
- **Error Formatting**: Proper error field structure for GCP error reporting

## GCP Cloud Logging Fields

The handler automatically formats logs with these GCP-recognized fields:

- `severity`: Log level (DEBUG, INFO, WARNING, ERROR)
- `message`: Log message
- `timestamp`: RFC3339 formatted timestamp
- `logging.googleapis.com/sourceLocation`: Source code location
- `logging.googleapis.com/trace`: Trace ID for distributed tracing
- `logging.googleapis.com/spanId`: Span ID for distributed tracing
- `httpRequest`: HTTP request details (when using middleware)

## Usage

### Basic Setup

```go
import (
    "log/slog"
    "os"
    "github.com/aliuygur/n8n-saas-api/internal/gcplog"
)

func main() {
    // Create GCP-optimized logger
    logger := slog.New(gcplog.NewHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)

    // Use logger
    logger.Info("Application started")
    logger.Error("Something went wrong", "error", err)
}
```

### Environment-based Setup (Recommended)

The application automatically selects the appropriate log handler based on the `ENV` environment variable:

- **Development** (`ENV=development` or not set): Uses `slog.TextHandler` for human-readable output
- **Production** (`ENV=production`): Uses `gcplog.Handler` for GCP Cloud Logging

```go
import (
    "log/slog"
    "os"
    "github.com/aliuygur/n8n-saas-api/internal/config"
    "github.com/aliuygur/n8n-saas-api/internal/gcplog"
)

func main() {
    cfg, _ := config.Load()

    var logger *slog.Logger
    if cfg.Server.IsDevelopment() {
        logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
    } else {
        logger = slog.New(gcplog.NewHandler(os.Stdout, nil))
    }
    slog.SetDefault(logger)
}
```

**Local development output** (TextHandler):
```
time=2024-01-15T10:30:45.123-07:00 level=INFO msg="Application started" env=development
time=2024-01-15T10:30:46.456-07:00 level=INFO msg="HTTP request" path=/api/users status=200
```

**Production output** (GCP Handler):
```json
{"severity":"INFO","message":"Application started","timestamp":"2024-01-15T10:30:45.123Z","env":"production"}
{"severity":"INFO","message":"HTTP request","timestamp":"2024-01-15T10:30:46.456Z","path":"/api/users","status":200}
```

### HTTP Request Logging

```go
import (
    "net/http"
    "github.com/aliuygur/n8n-saas-api/internal/gcplog"
)

func main() {
    logger := slog.New(gcplog.NewHandler(os.Stdout, nil))

    mux := http.NewServeMux()
    mux.HandleFunc("/", handler)

    // Wrap with GCP HTTP logging middleware
    handler := gcplog.HTTPMiddleware(logger)(mux)

    http.ListenAndServe(":8080", handler)
}
```

### Distributed Tracing

When using Cloud Trace, include trace and span IDs in your logs:

```go
logger.Info("Processing request",
    "trace", "projects/PROJECT_ID/traces/TRACE_ID",
    "span", "SPAN_ID",
    "user_id", userID,
)
```

### Error Logging

Errors are automatically formatted for GCP Error Reporting:

```go
if err != nil {
    logger.Error("Database query failed",
        "error", err,
        "query", sqlQuery,
        "user_id", userID,
    )
}
```

## Output Example

```json
{
  "severity": "INFO",
  "message": "HTTP request",
  "timestamp": "2024-01-15T10:30:45.123Z",
  "logging.googleapis.com/sourceLocation": {
    "file": "/app/main.go",
    "line": "42",
    "function": "main.handleRequest"
  },
  "httpRequest": {
    "requestMethod": "GET",
    "requestUrl": "/api/users",
    "status": 200,
    "responseSize": "1234",
    "userAgent": "Mozilla/5.0...",
    "remoteIp": "192.168.1.1",
    "latency": "45.123ms",
    "protocol": "HTTP/1.1"
  },
  "path": "/api/users",
  "method": "GET",
  "status": 200,
  "duration_ms": 45
}
```

## GKE Autopilot Configuration

When running on GKE Autopilot, Cloud Logging automatically collects logs from stdout/stderr. No additional configuration needed:

1. Logs are automatically sent to Cloud Logging
2. Log entries appear in the Logs Explorer
3. Error Reporting automatically detects errors
4. Trace integration works automatically when trace IDs are included

## Best Practices

1. **Use Structured Fields**: Always add context as key-value pairs, not in the message
   ```go
   // Good
   logger.Info("User logged in", "user_id", userID, "ip", ip)

   // Bad
   logger.Info(fmt.Sprintf("User %s logged in from %s", userID, ip))
   ```

2. **Include Trace IDs**: For distributed systems, always include trace and span IDs
   ```go
   logger.Info("API call", "trace", traceID, "span", spanID)
   ```

3. **Use Appropriate Levels**:
   - `DEBUG`: Development/debugging information
   - `INFO`: General informational messages
   - `WARNING`: Warning messages (potential issues)
   - `ERROR`: Error messages (actual failures)

4. **Add Context**: Include relevant context fields for better log analysis
   ```go
   logger.Error("Payment failed",
       "error", err,
       "user_id", userID,
       "amount", amount,
       "payment_method", method,
   )
   ```

## Integration with GCP Services

### Cloud Trace
Include trace information in logs for correlation:
```go
logger.Info("Database query",
    "trace", fmt.Sprintf("projects/%s/traces/%s", projectID, traceID),
    "query_duration_ms", duration.Milliseconds(),
)
```

### Error Reporting
Errors logged at ERROR level are automatically sent to Error Reporting when the error field is included.

### Logs Explorer
Use these queries in Logs Explorer to filter logs:
- By severity: `severity="ERROR"`
- By HTTP status: `httpRequest.status>=500`
- By trace: `trace="projects/PROJECT_ID/traces/TRACE_ID"`

## Performance

The handler is designed for production use:
- Minimal allocations
- Efficient JSON marshaling
- No external dependencies beyond standard library

## References

- [GCP Cloud Logging Documentation](https://cloud.google.com/logging/docs)
- [LogEntry Format](https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry)
- [httpRequest Format](https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#HttpRequest)
