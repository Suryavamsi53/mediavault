package entity

import (
	"time"
)

// User represents a user account.
type User struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"username" db:"username"`
	Password  string    `json:"-" db:"password"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TableName specifies the database table for User.
func (User) TableName() string {
	return "users"
}

// GetID returns the user ID.
func (u User) GetID() string {
	return u.ID
}

// GetName returns the user name.
func (u User) GetName() string {
	return u.Name
}
