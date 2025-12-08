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
//   encore secret set PolarAccessToken --dev
//   encore secret set PolarProductID --dev
//   encore secret set PolarWebhookSecret --dev
// Get the webhook secret from Polar dashboard > Settings > Webhooks
var secrets struct {
	PolarAccessToken   string
	PolarProductID     string
	PolarWebhookSecret string
}

// initService initializes the subscription service
func initService() (*Service, error) {
	// Initialize Polar client with sandbox mode for testing
	polarClient := polargo.New(
		polargo.WithServer("sandbox"), // Use "production" when ready
		polargo.WithSecurity(secrets.PolarAccessToken),
	)

	return &Service{
		db:          subscriptionDB.Stdlib(),
		polarClient: polarClient,
	}, nil
}
