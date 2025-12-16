package gcplog_test

import (
	"bytes"
	"errors"
	"log/slog"

	"github.com/aliuygur/n8n-saas-api/internal/gcplog"
)

func ExampleNewHandler() {
	var buf bytes.Buffer
	logger := slog.New(gcplog.NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Basic logging
	logger.Info("Application started")

	// Logging with attributes
	logger.Info("User action",
		"user_id", "12345",
		"action", "login",
		"ip", "192.168.1.1",
	)

	// Error logging
	err := errors.New("connection timeout")
	logger.Error("Database connection failed",
		"error", err,
		"database", "postgres",
		"retry_count", 3,
	)

	// With trace information
	logger.Info("API request processed",
		"trace", "projects/my-project/traces/abcd1234",
		"span", "span123",
		"duration_ms", 45,
	)
}

func ExampleNewHandler_grouping() {
	var buf bytes.Buffer
	logger := slog.New(gcplog.NewHandler(&buf, nil))

	// Grouped attributes
	logger.WithGroup("user").Info("Profile updated",
		"id", "12345",
		"name", "John Doe",
		"email", "john@example.com",
	)
}
