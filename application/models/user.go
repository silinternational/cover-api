package models

import (
	"net/http"
	"time"

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
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
//  It first adds a UUID to the user if its UUID is empty
func (u *User) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(u), nil
}

func (u *User) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(u, id)
}

func (u *User) IsActorAllowedTo(actor User, p Permission, subResource string, req *http.Request) bool {
	switch p {
	case PermissionView:
		return true
	case PermissionList, PermissionCreate, PermissionDelete:
		return actor.IsAdmin()
	case PermissionUpdate:
		return actor.IsAdmin() || actor.ID.String() == u.ID.String()
	default:
		return false
	}
}

func (u *User) IsAdmin() bool {
	return false
}
