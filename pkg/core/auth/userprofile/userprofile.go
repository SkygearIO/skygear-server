package userprofile

import (
	"time"
)

// Data refers the profile info of a user,
// like username, email, age, phone number
type Data map[string]interface{}

// UserProfile refers user profile data type
type UserProfile struct {
	ID        string
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
	Data
}
