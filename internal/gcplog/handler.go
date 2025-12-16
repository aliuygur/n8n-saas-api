package gcplog

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"runtime"
	"strconv"
)

// GCPHandler is a slog.Handler that formats logs for GCP Cloud Logging
type GCPHandler struct {
	w      io.Writer
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

// NewHandler creates a new GCP Cloud Logging compatible handler
func NewHandler(w io.Writer, opts *slog.HandlerOptions) *GCPHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	level := slog.LevelInfo
	if opts.Level != nil {
		level = opts.Level.Level()
	}
	return &GCPHandler{
		w:     w,
		level: level,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *GCPHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats and writes a log record
func (h *GCPHandler) Handle(_ context.Context, r slog.Record) error {
	entry := make(map[string]interface{})

	// GCP Cloud Logging severity field
	entry["severity"] = severityFromLevel(r.Level)

	// Standard message field
	entry["message"] = r.Message

	// Timestamp in RFC3339 format (GCP preferred)
	entry["timestamp"] = r.Time.Format("2006-01-02T15:04:05.000Z07:00")

	// Add source location if available
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		entry["logging.googleapis.com/sourceLocation"] = map[string]interface{}{
			"file":     f.File,
			"line":     strconv.Itoa(f.Line),
			"function": f.Function,
		}
	}

	// Add handler-level attributes
	for _, attr := range h.attrs {
		addAttr(entry, attr, h.groups)
	}

	// Add record attributes
	r.Attrs(func(a slog.Attr) bool {
		addAttr(entry, a, h.groups)
		return true
	})

	// Write JSON to output
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = h.w.Write(data)
	return err
}

// WithAttrs returns a new handler with additional attributes
func (h *GCPHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &GCPHandler{
		w:      h.w,
		level:  h.level,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

// WithGroup returns a new handler with a group name prepended to attributes
func (h *GCPHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &GCPHandler{
		w:      h.w,
		level:  h.level,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// severityFromLevel maps slog levels to GCP Cloud Logging severity levels
func severityFromLevel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERROR"
	case level >= slog.LevelWarn:
		return "WARNING"
	case level >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// addAttr adds an attribute to the entry map, respecting groups
func addAttr(entry map[string]interface{}, attr slog.Attr, groups []string) {
	key := attr.Key
	value := attr.Value.Any()

	// Special handling for common GCP fields
	switch key {
	case "error", "err":
		// GCP expects error information in a specific format
		if err, ok := value.(error); ok {
			entry["error"] = map[string]interface{}{
				"message": err.Error(),
			}
			return
		}
	case "trace", "trace_id":
		// GCP trace field
		entry["logging.googleapis.com/trace"] = value
		return
	case "span", "span_id":
		// GCP span field
		entry["logging.googleapis.com/spanId"] = value
		return
	case "httpRequest":
		// GCP httpRequest field - pass through as-is
		entry["httpRequest"] = value
		return
	}

	// Handle grouped attributes
	if len(groups) > 0 {
		// Navigate/create nested structure
		current := entry
		for i, group := range groups {
			if i == len(groups)-1 {
				// Last group - add the value here
				if _, exists := current[group]; !exists {
					current[group] = make(map[string]interface{})
				}
				if groupMap, ok := current[group].(map[string]interface{}); ok {
					groupMap[key] = value
				}
			} else {
				// Intermediate group - ensure it exists
				if _, exists := current[group]; !exists {
					current[group] = make(map[string]interface{})
				}
				if groupMap, ok := current[group].(map[string]interface{}); ok {
					current = groupMap
				}
			}
		}
	} else {
		// No groups - add directly to entry
		entry[key] = value
	}
}
