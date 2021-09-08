package models

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/auth"
	"github.com/silinternational/cover-api/domain"
)

type UserAppRole string

const (
	AppRoleAdmin    = UserAppRole("Admin")
	AppRoleSteward  = UserAppRole("Steward")
	AppRoleSignator = UserAppRole("Signator")
	AppRoleUser     = UserAppRole("User")
)

var validUserAppRoles = map[UserAppRole]struct{}{
	AppRoleAdmin:    {},
	AppRoleSteward:  {},
	AppRoleSignator: {},
	AppRoleUser:     {},
}

// Users is a slice of User objects
type Users []User

// User model
type User struct {
	ID            uuid.UUID   `json:"-" db:"id"`
	Email         string      `db:"email" validate:"required"`
	EmailOverride string      `db:"email_override"`
	FirstName     string      `db:"first_name"`
	LastName      string      `db:"last_name"`
	IsBlocked     bool        `db:"is_blocked"`
	LastLoginUTC  time.Time   `db:"last_login_utc"`
	Location      string      `db:"location"`
	StaffID       string      `db:"staff_id"`
	AppRole       UserAppRole `db:"app_role" validate:"appRole"`
	PhotoFileID   nulls.UUID  `json:"photo_file_id" db:"photo_file_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	Policies Policies `many_to_many:"policy_users"`

	// File object that contains the user's avatar or photo
	PhotoFile *File `belongs_to:"files"`
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

func (u *User) FindByEmail(tx *pop.Connection, email string) error {
	return tx.Where("email = ?", email).First(u)
}

func (u *User) FindByStaffID(tx *pop.Connection, id string) error {
	return tx.Where("staff_id = ?", id).First(u)
}

func (u *User) IsActorAllowedTo(tx *pop.Connection, actor User, p Permission, sub SubResource, req *http.Request) bool {
	switch p {
	case PermissionView:
		return actor.IsAdmin() || actor.ID.String() == u.ID.String()
	case PermissionList, PermissionCreate, PermissionDelete:
		return actor.IsAdmin()
	case PermissionUpdate:
		return actor.IsAdmin() || actor.ID.String() == u.ID.String()
	default:
		return false
	}
}

func (u *User) IsAdmin() bool {
	return u.AppRole == AppRoleAdmin || u.AppRole == AppRoleSteward || u.AppRole == AppRoleSignator
}

func (u *User) FindOrCreateFromAuthUser(tx *pop.Connection, authUser *auth.User) error {
	isNewUser := false

	// Try finding user by StaffID first and otherwise by Email
	if err := u.FindByStaffID(tx, authUser.StaffID); err != nil {
		if domain.IsOtherThanNoRows(err) {
			return err
		}

		if err := u.FindByEmail(tx, authUser.Email); err != nil {
			if domain.IsOtherThanNoRows(err) {
				return err
			}
			isNewUser = true
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

	// If this is a brand-new user, create a Policy for them
	if !isNewUser {
		return nil
	}
	e := events.Event{
		Kind:    domain.EventApiUserCreated,
		Message: fmt.Sprintf("Username: %s %s  ID: %s", u.FirstName, u.LastName, u.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: u.ID},
	}
	emitEvent(e)

	return nil
}

func (u *User) EmailOfChoice() string {
	if u.EmailOverride != "" {
		return u.EmailOverride
	}
	return u.Email
}

func (u *User) FindSteward(tx *pop.Connection) {
	if err := tx.Where("app_role = ?", AppRoleSteward).First(u); err != nil {
		panic("error finding steward user" + err.Error())
	}
}

func (u *User) FindSignator(tx *pop.Connection) {
	if err := tx.Where("app_role = ?", AppRoleSignator).First(u); err != nil {
		panic("error finding signator user " + err.Error())
	}
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

func (u *User) LoadPolicies(tx *pop.Connection, reload bool) {
	if len(u.Policies) == 0 || reload {
		if err := tx.Load(u, "Policies"); err != nil {
			panic("database error loading User.Policies, " + err.Error())
		}
	}
}

func (u *User) MyClaims(tx *pop.Connection) Claims {
	if err := tx.Load(u, "Policies.Claims"); err != nil {
		panic("database error loading User.Policies.Claims, " + err.Error())
	}

	var claims Claims
	for _, policy := range u.Policies {
		claims = append(claims, policy.Claims...)
	}

	return claims
}

func (u *User) Name() string {
	return strings.TrimSpace(strings.TrimSpace(u.FirstName) + " " + strings.TrimSpace(u.LastName))
}

func (u *User) ConvertToPolicyMember() api.PolicyMember {
	return api.PolicyMember{
		ID:            u.ID,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Email:         u.Email,
		EmailOverride: u.EmailOverride,
		LastLoginUTC:  u.LastLoginUTC,
		Location:      u.Location,
	}
}

func (u *Users) ConvertToPolicyMembers() api.PolicyMembers {
	members := make(api.PolicyMembers, len(*u))
	for i, uu := range *u {
		members[i] = uu.ConvertToPolicyMember()
	}

	return members
}

// CreateInitialPolicy is a hack to create an initial policy for a new user
func (u *User) CreateInitialPolicy(tx *pop.Connection) error {
	if u == nil || u.ID == uuid.Nil {
		return errors.New("user must have an ID in CreateInitialPolicy")
	}
	if tx == nil {
		tx = DB
	}

	policy := Policy{
		Type:        api.PolicyTypeHousehold,
		CostCenter:  fmt.Sprintf("CC-%s-%s", u.FirstName, u.LastName),
		HouseholdID: nulls.NewString(fmt.Sprintf("HHID-%s-%s", u.FirstName, u.LastName)),
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

func (u *Users) GetAll(tx *pop.Connection) error {
	return tx.All(u)
}

// OwnsFile returns true if the user owns the file identified by the given ID
func (u *User) OwnsFile(tx *pop.Connection, fileID uuid.UUID) (bool, error) {
	if u.ID == uuid.Nil {
		return false, errors.New("no user ID provided")
	}
	var f File
	err := tx.Find(&f, fileID)
	if err != nil {
		if domain.IsOtherThanNoRows(err) {
			panic("error finding file: " + err.Error())
		}

		return false, errors.New("unable to find file with ID : " + fileID.String())

	}
	return u.ID == f.CreatedByID, nil
}

// AttachPhotoFile assigns a previously-stored File to this User as a profile photo
func (u *User) AttachPhotoFile(tx *pop.Connection, fileID uuid.UUID) error {
	if err := addFile(tx, u, fileID); err != nil {
		return err
	}

	u.LoadPhotoFile(tx)
	return nil
}

// LoadPhotoFile - a simple wrapper method for loading members on the struct
func (u *User) LoadPhotoFile(tx *pop.Connection) {
	if !u.PhotoFileID.Valid {
		return
	}
	if err := tx.Load(u, "PhotoFile"); err != nil {
		panic("database error loading User.PhotoFile, " + err.Error())
	}
}

func (u *Users) ConvertToAPI(tx *pop.Connection) api.Users {
	out := make(api.Users, len(*u))
	for i, uu := range *u {
		out[i] = uu.ConvertToAPI(tx)
	}
	return out
}

func (u *User) ConvertToAPI(tx *pop.Connection) api.User {
	u.LoadPhotoFile(tx)

	// TODO: provide more than one policy
	var policyID nulls.UUID
	if len(u.Policies) > 0 {
		policyID = nulls.NewUUID(u.Policies[0].ID)
	}

	output := api.User{
		ID:            u.ID,
		Email:         u.Email,
		EmailOverride: u.EmailOverride,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Name:          u.Name(),
		LastLoginUTC:  u.LastLoginUTC,
		Location:      u.Location,
		PhotoFileID:   u.PhotoFileID,
		PolicyID:      policyID,
	}

	if u.PhotoFile != nil {
		f := u.PhotoFile.ConvertToAPI(tx)
		output.PhotoFile = &f
	}

	return output
}
