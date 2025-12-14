package types

import (
	"fmt"
	"time"
)

// Instance represents an instance for internal use (domain layer)
type Instance struct {
	ID         string
	UserID     string
	Status     string
	Namespace  string
	Subdomain  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeployedAt *time.Time
	DeletedAt  *time.Time
}

func (i *Instance) GetInstanceURL() string {
	return fmt.Sprintf("https://%s.instol.cloud", i.Subdomain)
}
