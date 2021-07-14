package models

import (
	"time"

	"github.com/gofrs/uuid"
)

// User model
type User struct {
	ID        int       `json:"-" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Email     string    `json:"email" db:"email"`
	UUID      uuid.UUID `json:"uuid" db:"uuid"`
}
