package components

// Instance represents an n8n instance
type Instance struct {
	ID          string
	InstanceURL string
	Status      string
	Subdomain   string
	CreatedAt   string
}

func (i *Instance) GetInstanceURL() string {
	return i.InstanceURL
}

// Subscription represents a user's subscription
type Subscription struct {
	ID                  string
	UserID              string
	PolarProductID      string
	PolarCustomerID     string
	PolarSubscriptionID string
	Status              string
	TrialEndsAt         string
	CreatedAt           string
	UpdatedAt           string
}
