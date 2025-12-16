package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appreq"
	"github.com/aliuygur/n8n-saas-api/internal/config"
	"github.com/aliuygur/n8n-saas-api/internal/gcplog"
	"github.com/aliuygur/n8n-saas-api/internal/handler"
	"github.com/aliuygur/n8n-saas-api/internal/services"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// Load configuration first to determine environment
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize logger based on environment
	var logger *slog.Logger
	if cfg.Server.IsDevelopment() {
		// Use human-readable text handler for local development
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	} else {
		// Use GCP Cloud Logging compatible handler for production
		logger = slog.New(gcplog.NewHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}
	slog.SetDefault(logger)

	logger.Info("Configuration loaded successfully", "env", cfg.Server.Env)

	// Connect to database
	db, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	logger.Info("Database connection established")

	// Initialize services
	svc, err := services.NewService(db, cfg)
	if err != nil {
		logger.Error("Failed to initialize services", "error", err)
		os.Exit(1)
	}

	// Initialize handler
	h, err := handler.New(cfg, svc)
	if err != nil {
		logger.Error("Failed to initialize handler", "error", err)
		os.Exit(1)
	}

	// Setup HTTP router
	mux := http.NewServeMux()

	// Register routes
	h.RegisterRoutes(mux)

	// Wrap mux with custom 404 handler
	handler := h.NotFoundHandlerWrapper(mux)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      appreq.Handler(handler, logger),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited gracefully")
}
