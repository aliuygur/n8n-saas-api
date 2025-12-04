package views

import "time"

// User represents a user in the system
type User struct {
	ID        string
	Email     string
	Password  string
	FirstName string
	LastName  string
	CreatedAt time.Time
}
