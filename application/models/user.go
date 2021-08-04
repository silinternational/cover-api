package models

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/silinternational/riskman-api/api"

	"github.com/gobuffalo/events"

	"github.com/silinternational/riskman-api/domain"

	"github.com/pkg/errors"
	"github.com/silinternational/riskman-api/auth"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

type UserAppRole string

const (
	AppRoleAdmin = UserAppRole("Admin")
	AppRoleUser  = UserAppRole("User")
)

var validUserAppRoles = map[UserAppRole]struct{}{
	AppRoleAdmin: {},
	AppRoleUser:  {},
}

// Users is a slice of User objects
type Users []User

// User model
type User struct {
	ID           uuid.UUID   `json:"-" db:"id"`
	Email        string      `db:"email" validate:"required"`
	FirstName    string      `db:"first_name"`
	LastName     string      `db:"last_name"`
	IsBlocked    bool        `db:"is_blocked"`
	LastLoginUTC time.Time   `db:"last_login_utc"`
	StaffID      string      `db:"staff_id"`
	AppRole      UserAppRole `db:"app_role" validate:"appRole"`
	CreatedAt    time.Time   `db:"created_at"`
	UpdatedAt    time.Time   `db:"updated_at"`

	Policies Policies `many_to_many:"policy_users"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
//  It first adds a UUID to the user if its UUID is empty
func (u *User) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(u), nil
}

// Create stores the User data as a new record in the database.
func (u *User) Create(tx *pop.Connection) error {
	return create(tx, u)
}

// Update writes the User data to an existing database record.
func (u *User) Update(tx *pop.Connection) error {
	return update(tx, u)
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

func (u *User) IsActorAllowedTo(tx *pop.Connection, actor User, p Permission, sub SubResource, req *http.Request) bool {
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
	return u.AppRole == AppRoleAdmin
}

func (u *User) FindOrCreateFromAuthUser(tx *pop.Connection, authUser *auth.User) error {
	isNewUser := false
	if err := u.FindByStaffID(tx, authUser.StaffID); err != nil {
		if domain.IsOtherThanNoRows(err) {
			return err
		}
		isNewUser = true
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

	// If this is a brand-new user, create a Policy for them
	if !isNewUser {
		return nil
	}
	e := events.Event{
		Kind:    domain.EventApiUserCreated,
		Message: fmt.Sprintf("Username: %s %s  ID: %s", u.FirstName, u.LastName, u.ID.String()),
		Payload: events.Payload{"id": u.ID.String()},
	}
	emitEvent(e)

	return nil
}

// CreateAccessToken - Create and store new UserAccessToken
func (u *User) CreateAccessToken(tx *pop.Connection, clientID string) (UserAccessToken, error) {
	if clientID == "" {
		return UserAccessToken{}, fmt.Errorf(
			"cannot create token with empty clientID for user %s %s", u.FirstName, u.LastName)
	}

	uat := InitAccessToken(clientID)
	uat.UserID = u.ID

	if err := uat.Create(tx); err != nil {
		return uat, fmt.Errorf("error creating user access token id: %s ... %s", u.ID, err)
	}

	return uat, nil
}

func (u *User) LoadPolicies(tx *pop.Connection, reload bool) error {
	if len(u.Policies) == 0 || reload {
		return tx.Load(u, "Policies")
	}
	return nil
}

func ConvertPolicyMember(tx *pop.Connection, u User) (api.PolicyMember, error) {
	return api.PolicyMember{
		ID:           u.ID,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		Email:        u.Email,
		LastLoginUTC: u.LastLoginUTC,
	}, nil
}

func ConvertPolicyMembers(tx *pop.Connection, us Users) (api.PolicyMembers, error) {
	members := make(api.PolicyMembers, len(us))
	for i, u := range us {
		var err error
		members[i], err = ConvertPolicyMember(tx, u)
		if err != nil {
			return api.PolicyMembers{}, err
		}
	}

	return members, nil
}

// CreateInitialPolicy is a hack to create an initial policy for a new user
func (u *User) CreateInitialPolicy(tx *pop.Connection) error {
	if u.ID == uuid.Nil {
		return errors.New("user must have an ID in CreateInitialPolicy")
	}
	if tx == nil {
		tx = DB
	}

	policy := Policy{
		Type:        api.PolicyTypeHousehold,
		CostCenter:  fmt.Sprintf("CC-%s-%s", u.FirstName, u.LastName),
		HouseholdID: fmt.Sprintf("HHID-%s-%s", u.FirstName, u.LastName),
	}

	if err := tx.Create(&policy); err != nil {
		return errors.New("unable to create initial policy in CreateInitialPolicy: " + err.Error())
	}

	polUser := PolicyUser{
		PolicyID: policy.ID,
		UserID:   u.ID,
	}

	if err := tx.Create(&polUser); err != nil {
		return errors.New("unable to create policy-user in CreateInitialPolicy: " + err.Error())
	}
	return nil
}
