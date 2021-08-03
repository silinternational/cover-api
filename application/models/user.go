package models

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/silinternational/riskman-api/domain"

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

func (u *User) FindByStaffID(tx *pop.Connection, id string) error {
	return tx.Where("staff_id = ?", id).First(u)
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
	if err := u.FindByStaffID(tx, authUser.StaffID); err != nil {
		if domain.IsOtherThanNoRows(err) {
			return err
		}
	}

	// update attributes from authUser
	u.FirstName = authUser.FirstName
	u.LastName = authUser.LastName
	u.Email = authUser.Email
	u.StaffID = authUser.StaffID
	u.LastLoginUTC = time.Now().UTC()

	if err := tx.Save(u); err != nil {
		return errors.New("unable to save user record: " + err.Error())
	}

	return nil
}

// CreateAccessToken - Create and store new UserAccessToken
func (u *User) CreateAccessToken(tx *pop.Connection, clientID string) (string, int64, error) {
	if clientID == "" {
		return "", 0, fmt.Errorf("cannot create token with empty clientID for user %s", u.Nickname)
	}

	token, _ := getRandomToken()
	hash := HashClientIdAccessToken(clientID + token)
	expireAt := createAccessTokenExpiry()

	userAccessToken := &UserAccessToken{
		UserID:      u.ID,
		AccessToken: hash,
		ExpiresAt:   expireAt,
	}

	if err := userAccessToken.Create(tx); err != nil {
		return "", 0, err
	}

	return token, expireAt.UTC().Unix(), nil
}
