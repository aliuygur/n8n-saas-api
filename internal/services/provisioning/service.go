package provisioning

import (
	"database/sql"
	"fmt"

	"github.com/aliuygur/n8n-saas-api/internal/cloudflare"
	"github.com/aliuygur/n8n-saas-api/internal/gke"

	"encore.dev/storage/sqldb"
)

// Encore database definition
var mainDB = sqldb.NewDatabase("main", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

//encore:service
type Service struct {
	db         *sql.DB
	gke        *gke.Client
	cloudflare *cloudflare.Client
	config     Config
}

type Config struct {
	DefaultProjectID   string
	DefaultZone        string
	DefaultClusterName string
	CredentialsJSON    []byte
}

// Encore magic service initialization function
func initService() (*Service, error) {
	// TODO: Configure these values through Encore secrets or environment variables
	config := Config{
		DefaultProjectID:   "rockads",
		DefaultZone:        "europe-west1",
		DefaultClusterName: "autopilot-cluster-1",
	}

	gkeClient, err := gke.NewClient(config.DefaultProjectID, []byte(secrets.GCPCredentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create GKE client: %w", err)
	}

	cloudflareConfig := cloudflare.Config{
		APIToken:  secrets.CLOUDFLARE_API_TOKEN,
		TunnelID:  "a8486899-cc12-4466-a033-6f01a6a9e6d7",
		AccountID: "0f2a166551aa3c5afa61935e17a188e5",
		ZoneID:    "e5e4c6fce9052cf8823c291c54d64b51", // Your zone ID as a variable
	}
	cloudflareClient := cloudflare.NewClient(cloudflareConfig)

	return &Service{
		db:         mainDB.Stdlib(),
		gke:        gkeClient,
		cloudflare: cloudflareClient,
		config:     config,
	}, nil
}

var secrets struct {
	GCPCredentials       string
	CLOUDFLARE_API_TOKEN string
}
