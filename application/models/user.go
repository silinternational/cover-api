package models

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// Users is a slice of User objects
type Users []User

// User model
type User struct {
	ID           uuid.UUID `json:"-" db:"id"`
	Email        string    `db:"email" validate:"required"`
	FirstName    string    `db:"first_name"`
	LastName     string    `db:"last_name"`
	IsBlocked    bool      `db:"is_blocked"`
	LastLoginUTC time.Time `db:"last_login_utc"`
	StaffID      string    `db:"staff_id"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
//  It first adds a UUID to the user if its UUID is empty
func (u *User) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(u), nil
}

// HashClientIdAccessToken just returns a sha256.Sum256 of the input value
func HashClientIdAccessToken(accessToken string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(accessToken)))
}
