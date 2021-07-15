package models

import (
	"time"

	"github.com/gofrs/uuid"
)

// User model
type User struct {
	ID        int       `json:"-" db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Email     string    `db:"email"`
	UUID      uuid.UUID `db:"uuid"`
}
