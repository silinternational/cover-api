package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/silinternational/cover-api/api"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
)

// UserAccessToken is used by pop to map your user_access_tokens database table to your go code.
type UserAccessToken struct {
	ID          uuid.UUID  `db:"id"`
	UserID      uuid.UUID  `db:"user_id" validate:"required"`
	AccessToken string     `db:"-"`
	TokenHash   string     `db:"access_token" validate:"required"`
	ExpiresAt   time.Time  `db:"expires_at" validate:"required"`
	LastUsedAt  nulls.Time `db:"last_used_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`

	User *User `belongs_to:"users"`
}

// String is not required by pop and may be deleted
func (u UserAccessToken) String() string {
	ju, _ := json.Marshal(u)
	return string(ju)
}

// UserAccessTokens is not required by pop and may be deleted
type UserAccessTokens []UserAccessToken

// String is not required by pop and may be deleted
func (u UserAccessTokens) String() string {
	ju, _ := json.Marshal(u)
	return string(ju)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
// This method is not required and may be deleted.
func (u *UserAccessToken) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(u), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (u *UserAccessToken) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (u *UserAccessToken) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// DeleteByAccessToken uses a sha256.Sum256 of the accessToken to find which UserAccessToken to delete
// returns an api.AppError
func (u *UserAccessToken) DeleteByAccessToken(tx *pop.Connection, token string) error {
	if appErr := u.FindByAccessToken(tx, token); appErr != nil {
		return appErr
	}
	if err := u.Destroy(tx); err != nil {
		panic("database error trying to destroy user access token: " + err.Error())
	}

	return nil
}

// DeleteIfExpired checks the token expiration and returns `true` if expired. Also deletes
// the token from the database if it is expired.
func (u *UserAccessToken) DeleteIfExpired(tx *pop.Connection) (bool, error) {
	if u.ExpiresAt.Before(time.Now()) {
		err := u.Destroy(tx)
		if err != nil {
			return true, fmt.Errorf("unable to delete expired userAccessToken, id: %v", u.ID)
		}
		return true, nil
	}
	return false, nil
}

func (u *UserAccessToken) Destroy(tx *pop.Connection) error {
	return tx.Destroy(u)
}

// FindByAccessToken uses a sha256.Sum256 of the accessToken to find the corresponding UserAccessToken
// returns an api.AppError
func (u *UserAccessToken) FindByAccessToken(tx *pop.Connection, token string) error {
	if err := tx.Eager().Where("access_token = ?", HashAccessToken(token)).First(u); err != nil {
		l := len(token)
		if l > 5 {
			l = 5
		}

		if domain.IsOtherThanNoRows(err) {
			panic("database error trying to find user access token: " + err.Error())
		}

		appErr := api.AppError{
			Err:      err,
			Key:      api.ErrorFindingAccessToken,
			Category: api.CategoryUser,
			Message:  fmt.Sprintf("failed to find access token '%s...'", token[0:l]),
		}
		return &appErr
	}

	return nil
}

// GetUser returns the User associated with this access token
func (u *UserAccessToken) GetUser(tx *pop.Connection) (User, error) {
	if err := tx.Load(u, "User"); err != nil {
		return User{}, err
	}
	if u.User.Email == "" {
		return User{}, errors.New("no user associated with access token")
	}
	return *u.User, nil
}

func createAccessTokenExpiry() time.Time {
	dtNow := time.Now()
	return dtNow.Add(time.Second * time.Duration(domain.Env.AccessTokenLifetimeSeconds))
}

// Create stores the UserAccessToken data as a new record in the database.
func (u *UserAccessToken) Create(tx *pop.Connection) error {
	return create(tx, u)
}

func (u *UserAccessToken) Save(tx *pop.Connection) error {
	return save(tx, u)
}

// Update updates the UserAccessToken data in the database.
func (u *UserAccessToken) Update(tx *pop.Connection) error {
	return update(tx, u)
}

// InitAccessToken prepares a new value for the AccessToken field and the ExpiresAt field.
func InitAccessToken() UserAccessToken {
	token, _ := getRandomToken() // The init() function would have made sure there was no error

	if domain.Env.GoEnv == domain.EnvDevelopment {
		fmt.Printf("\n\ntoken: %s\n", token)
	}

	return UserAccessToken{
		AccessToken: token,
		TokenHash:   HashAccessToken(token),
		ExpiresAt:   createAccessTokenExpiry(),
	}
}
