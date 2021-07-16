package models

import (
	"time"

	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// Users is a slice of User objects
type Users []User

// User model
type User struct {
	ID        uuid.UUID `json:"-" db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Email     string    `db:"email" validate:"required"`
	UUID      uuid.UUID `db:"uuid" validate:"required"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
//  It first adds a UUID to the user if its UUID is empty
func (u *User) Validate(tx *pop.Connection) (*validate.Errors, error) {
	emptyUUID := uuid.UUID{}
	if u.UUID.String() == emptyUUID.String() {
		u.UUID = domain.GetUUID()
	}
	return validateModel(u), nil
}
