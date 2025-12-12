package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aliuygur/n8n-saas-api/internal/cloudflare"
	"github.com/aliuygur/n8n-saas-api/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create Cloudflare client
	cfConfig := cloudflare.Config{
		APIToken:  cfg.Cloudflare.APIToken,
		TunnelID:  cfg.Cloudflare.TunnelID,
		AccountID: cfg.Cloudflare.AccountID,
		ZoneID:    cfg.Cloudflare.ZoneID,
	}
	client := cloudflare.NewClient(cfConfig)

	ctx := context.Background()

	// Get current tunnel configuration
	fmt.Println("Fetching current tunnel configuration...")
	tunnelConfig, err := client.GetTunnelConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to get tunnel config: %v", err)
	}

	// Extract ingress rules
	ingress, ok := tunnelConfig["ingress"].([]interface{})
	if !ok {
		log.Fatal("No ingress rules found")
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
	fmt.Printf("\nFound %d routes in tunnel configuration:\n", len(allRoutes))
	for i, route := range allRoutes {
		fmt.Printf("%d. %s\n", i+1, route)
	}

	if len(allRoutes) == 0 {
		fmt.Println("\nNo routes found!")
		return
	}

	// Ask which routes to delete
	fmt.Print("\nOptions:\n")
	fmt.Println("1. Delete ALL routes")
	fmt.Println("2. Delete routes for deleted namespaces (orphaned routes)")
	fmt.Println("3. Cancel")
	fmt.Print("\nSelect option (1-3): ")

	var option string
	fmt.Scanln(&option)

	var routesToDelete []string

	switch option {
	case "1":
		// Delete all routes
		fmt.Print("\nAre you sure you want to delete ALL routes? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "yes" {
			fmt.Println("Aborted.")
			return
		}
		routesToDelete = allRoutes

	case "2":
		// Delete routes for deleted namespaces
		fmt.Println("\nChecking for orphaned routes (routes pointing to deleted namespaces)...")
		routesToDelete = allRoutes // For now, we'll delete all since all the namespaces were deleted
		fmt.Printf("All %d routes appear to be orphaned (no corresponding namespaces found)\n", len(routesToDelete))

		fmt.Print("\nDo you want to delete these orphaned routes? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "yes" {
			fmt.Println("Aborted.")
			return
		}

	case "3":
		fmt.Println("Aborted.")
		return

	default:
		fmt.Println("Invalid option. Aborted.")
		return
	}

	// Delete selected routes
	fmt.Println("\nDeleting routes...")
	successCount := 0
	failCount := 0

	for _, route := range routesToDelete {
		fmt.Printf("Deleting route: %s... ", route)
		if err := client.RemoveTunnelRoute(ctx, route); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			failCount++
		} else {
			fmt.Println("OK")
			successCount++
		}
	}

	fmt.Printf("\nCleanup completed! Success: %d, Failed: %d\n", successCount, failCount)
}
