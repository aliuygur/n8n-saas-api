package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/cloudflare"
	"github.com/aliuygur/n8n-saas-api/internal/config"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/services"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	// Parse command-line flags
	autoConfirm := flag.Bool("y", false, "Automatically confirm deletion without prompting")
	flag.Parse()

	// Load .env file
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	database, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create context with logger
	ctx := context.Background()
	ctx = appctx.WithLogger(ctx, logger)

	// Test database connection
	if err := database.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize services
	svc, err := services.NewService(database, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Fetch all instances
	queries := db.New(database)
	instances, err := queries.ListAllInstances(ctx, db.ListAllInstancesParams{
		Limit:  1000, // Fetch up to 1000 instances
		Offset: 0,
	})
	if err != nil {
		log.Fatalf("Failed to fetch instances: %v", err)
	}

	// Display all instances
	fmt.Printf("\nFound %d instances:\n", len(instances))
	for i, inst := range instances {
		fmt.Printf("%d. ID: %s, UserID: %s, Subdomain: %s, Namespace: %s, Status: %s\n",
			i+1, inst.ID, inst.UserID, inst.Subdomain, inst.Namespace, inst.Status)
	}

	if len(instances) == 0 {
		fmt.Println("\nNo instances found!")
	} else {
		// Ask for confirmation (unless auto-confirm is enabled)
		if !*autoConfirm {
			fmt.Print("\nAre you sure you want to delete ALL instances? This will:\n")
			fmt.Println("  - Delete Kubernetes namespaces")
			fmt.Println("  - Delete DNS records from Cloudflare")
			fmt.Println("  - Mark instances as deleted in database")
			fmt.Print("\nType 'yes' to confirm: ")

			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "yes" {
				fmt.Println("Aborted.")
				return
			}
		} else {
			fmt.Println("\nAuto-confirm enabled. Proceeding with deletion...")
		}

		// Delete instances
		fmt.Println("\nDeleting instances...")
		successCount := 0
		failCount := 0

		for _, inst := range instances {
			fmt.Printf("Deleting instance: %s (subdomain: %s)... ", inst.ID, inst.Subdomain)

			err := svc.DeleteInstance(ctx, services.DeleteInstanceParams{
				UserID:     inst.UserID,
				InstanceID: inst.ID,
			})

			if err != nil {
				fmt.Printf("FAILED: %v\n", err)
				failCount++
			} else {
				fmt.Println("OK")
				successCount++
			}
		}

		fmt.Printf("\nInstance cleanup completed! Success: %d, Failed: %d\n", successCount, failCount)
	}

	// Now cleanup orphaned tunnel routes
	fmt.Println("\n=== Cleaning up orphaned tunnel routes ===")
	cleanupTunnelRoutes(ctx, cfg)
}

func cleanupTunnelRoutes(ctx context.Context, cfg *config.Config) {
	// Create Cloudflare client
	cfConfig := cloudflare.Config{
		APIToken:  cfg.Cloudflare.APIToken,
		TunnelID:  cfg.Cloudflare.TunnelID,
		AccountID: cfg.Cloudflare.AccountID,
		ZoneID:    cfg.Cloudflare.ZoneID,
	}
	client := cloudflare.NewClient(cfConfig)

	// Get current tunnel configuration
	fmt.Println("Fetching current tunnel configuration...")
	tunnelConfig, err := client.GetTunnelConfig(ctx)
	if err != nil {
		log.Printf("Failed to get tunnel config: %v", err)
		return
	}

	// Extract ingress rules
	ingress, ok := tunnelConfig["ingress"].([]interface{})
	if !ok {
		fmt.Println("No ingress rules found")
		return
	}

	// Get all routes with hostnames (excluding the catch-all)
	allRoutes := []string{}
	for _, rule := range ingress {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		hostname, hasHostname := ruleMap["hostname"].(string)
		if !hasHostname || hostname == "" {
			continue
		}

		allRoutes = append(allRoutes, hostname)
	}

	// Display all routes
	fmt.Printf("\nFound %d tunnel routes:\n", len(allRoutes))
	for i, route := range allRoutes {
		fmt.Printf("%d. %s\n", i+1, route)
	}

	if len(allRoutes) == 0 {
		fmt.Println("\nNo tunnel routes found!")
		return
	}

	// Delete all tunnel routes
	fmt.Println("\nDeleting tunnel routes...")
	successCount := 0
	failCount := 0

	for _, route := range allRoutes {
		fmt.Printf("Deleting tunnel route: %s... ", route)
		if err := client.RemoveTunnelRoute(ctx, route); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			failCount++
		} else {
			fmt.Println("OK")
			successCount++
		}
	}

	fmt.Printf("\nTunnel cleanup completed! Success: %d, Failed: %d\n", successCount, failCount)
}
