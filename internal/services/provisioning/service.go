package provisioning

import (
	"database/sql"
	"fmt"

	"github.com/aliuygur/n8n-saas-api/internal/gke"

	"encore.dev/storage/sqldb"
)

// Encore database definition
var mainDB = sqldb.NewDatabase("main", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

//encore:service
type Service struct {
	db     *sql.DB
	gke    *gke.Client
	config Config
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
		DefaultProjectID:   "your-gcp-project-id",
		DefaultZone:        "us-central1-a",
		DefaultClusterName: "n8n-cluster",
		// CredentialsJSON will be loaded from Encore secrets
	}

	gkeClient, err := gke.NewClient(config.DefaultProjectID, config.CredentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create GKE client: %w", err)
	}

	return &Service{
		db:     mainDB.Stdlib(),
		gke:    gkeClient,
		config: config,
	}, nil
}
