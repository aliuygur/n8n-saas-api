package subscription

import (
	"database/sql"

	"encore.dev/storage/sqldb"
	polargo "github.com/polarsource/polar-go"
)

// Encore database definition
var subscriptionDB = sqldb.NewDatabase("subscription", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

//encore:service
type Service struct {
	db          *sql.DB
	polarClient *polargo.Polar
}

type Config struct {
	PolarAccessToken string
	PolarProductID   string
	PricePerInstance int64 // in cents
}

// TODO: Set secrets via Encore:
//   encore secret set POLAR_ACCESS_TOKEN --dev
//   encore secret set POLAR_PRODUCT_ID --dev
//   encore secret set POLAR_WEBHOOK_SECRET --dev
// Get the webhook secret from Polar dashboard > Settings > Webhooks
var secrets struct {
	POLAR_ACCESS_TOKEN   string
	POLAR_PRODUCT_ID     string
	POLAR_WEBHOOK_SECRET string
}

// initService initializes the subscription service
func initService() (*Service, error) {
	// Initialize Polar client with sandbox mode for testing
	polarClient := polargo.New(
		polargo.WithServer("sandbox"), // Use "production" when ready
		polargo.WithSecurity(secrets.POLAR_ACCESS_TOKEN),
	)

	return &Service{
		db:          subscriptionDB.Stdlib(),
		polarClient: polarClient,
	}, nil
}
