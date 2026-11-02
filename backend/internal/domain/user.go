package domain

import "time"

// User is an account: someone who can log in. It is not the same thing as a
// Participant — one user can be in many projects, and a participant (cast added
// by name) may never become a user at all.
type User struct {
	ID          string
	Email       string
	DisplayName string
	CreatedAt   time.Time
}
