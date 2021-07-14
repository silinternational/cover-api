package models

import (
	"time"

	"github.com/gofrs/uuid"
)

// User model
type User struct {
	ID        int       `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	UUID      uuid.UUID `json:"uuid"`
}
