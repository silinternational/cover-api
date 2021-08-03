package models

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/silinternational/riskman-api/auth"

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

func (u *User) GetID() uuid.UUID {
	return u.ID
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

func (u *User) FindOrCreateFromAuthUser(tx *pop.Connection, authUser *auth.User) error {
	newUser := true
	if u.ID != uuid.Nil {
		newUser = false
	}

	// update attributes from authUser
	u.FirstName = authUser.FirstName
	u.LastName = authUser.LastName
	u.Email = authUser.Email

	if err := u.Save(tx); err != nil {
		return errors.New("unable to save user record: " + err.Error())
	}

	return nil
}
