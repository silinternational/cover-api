package models

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/auth"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
	"github.com/silinternational/cover-api/storage"
)

type UserAppRole string

const (
	AppRoleCustomer = UserAppRole("Customer")
	AppRoleSignator = UserAppRole("Signator")
	AppRoleSteward  = UserAppRole("Steward")
	ServiceUserID   = "b9367e15-6a84-4fc8-b2ea-bd7615260b01"
)

var validUserAppRoles = map[UserAppRole]struct{}{
	AppRoleCustomer: {},
	AppRoleSignator: {},
	AppRoleSteward:  {},
}

// Users is a slice of User objects
type Users []User

// User model
type User struct {
	ID            uuid.UUID    `json:"-" db:"id"`
	Email         string       `db:"email" validate:"required"`
	EmailOverride string       `db:"email_override"`
	FirstName     string       `db:"first_name"`
	LastName      string       `db:"last_name"`
	IsBlocked     bool         `db:"is_blocked"`
	LastLoginUTC  time.Time    `db:"last_login_utc"`
	City          string       `db:"city"`
	State         string       `db:"state"`
	Country       string       `db:"country"`
	StaffID       nulls.String `db:"staff_id"`
	AppRole       UserAppRole  `db:"app_role" validate:"appRole"`
	PhotoFileID   nulls.UUID   `json:"photo_file_id" db:"photo_file_id"`

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
	if u.AppRole == "" {
		u.AppRole = AppRoleCustomer
	}
	return create(tx, u)
}

// Update writes the User data to an existing database record.
func (u *User) Update(tx *pop.Connection) error {
	if u.AppRole == "" {
		u.AppRole = AppRoleCustomer
	}
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
	return find(tx, u, id)
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

// IsAdmin returns true if the user has AppRole of Steward or Signator
func (u *User) IsAdmin() bool {
	return u.AppRole == AppRoleSteward || u.AppRole == AppRoleSignator
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

	if u.AppRole == "" {
		u.AppRole = AppRoleCustomer
		id, _ := strconv.Atoi(authUser.StaffID)
		if domain.Env.GoEnv == domain.EnvDevelopment || domain.Env.GoEnv == domain.EnvStaging {
			if id >= 1000000 {
				u.AppRole = AppRoleSteward
			}
			if id >= 5000000 {
				u.AppRole = AppRoleSignator
			}
		}
	}

	// update attributes from authUser
	u.FirstName = authUser.FirstName
	u.LastName = authUser.LastName
	u.Email = authUser.Email
	u.StaffID = nulls.NewString(authUser.StaffID)
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

// EmailOfChoice returns the user's EmailOverride value if it's not blank.
//   Otherwise it returns the user's Email value.
func (u *User) EmailOfChoice() string {
	if u.EmailOverride != "" {
		return u.EmailOverride
	}
	return u.Email
}

// GetPrimarySteward returns the User with AppRoleSteward and the earliest created_at
func GetPrimarySteward(tx *pop.Connection) User {
	u := User{}
	if err := tx.Where("app_role = ? AND NOT is_blocked", AppRoleSteward).Order("created_at ASC").First(&u); err != nil {
		log.Fatalf("error finding the primary steward user: %s", err)
	}
	return u
}

// GetServiceUser returns the Service User record
func GetServiceUser(tx *pop.Connection) User {
	u := User{}
	if err := tx.Find(&u, ServiceUserID); err != nil {
		log.Fatalf("error finding service user: %s", err)
	}
	return u
}

// FindStewards finds all the users with AppRoleSteward
func (u *Users) FindStewards(tx *pop.Connection) {
	if err := tx.Where("app_role = ?", AppRoleSteward).All(u); err != nil {
		panic("error finding steward users " + err.Error())
	}
}

// FindSignators finds all the users with AppRoleSignator
func (u *Users) FindSignators(tx *pop.Connection) {
	if err := tx.Where("app_role = ?", AppRoleSignator).All(u); err != nil {
		panic("error finding signator users " + err.Error())
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
	return u.GetName().String()
}

func (u *User) GetName() Name {
	return Name{
		First: u.FirstName,
		Last:  u.LastName,
	}
}

func (u *User) ConvertToPolicyMember(polUserID uuid.UUID) api.PolicyMember {
	return api.PolicyMember{
		ID:            u.ID,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Email:         u.Email,
		EmailOverride: u.EmailOverride,
		LastLoginUTC:  u.LastLoginUTC,
		Country:       u.GetLocation().Country,
		PolicyUserID:  polUserID,
	}
}

func (u *Users) ConvertToPolicyMembers(polUserIDs []uuid.UUID) api.PolicyMembers {
	userCount := len(*u)
	puCount := len(polUserIDs)
	if puCount != userCount {
		panic(fmt.Sprintf("number of polUserIDs: %d must match number of users: %d", puCount, userCount))
	}
	members := make(api.PolicyMembers, userCount)
	for i, uu := range *u {
		members[i] = uu.ConvertToPolicyMember(polUserIDs[i])
	}

	return members
}

// CreateInitialPolicy creates an initial policy for a new user
func (u *User) CreateInitialPolicy(tx *pop.Connection, householdID string) error {
	if u == nil || u.ID == uuid.Nil {
		return errors.New("user must have an ID in CreateInitialPolicy")
	}
	if tx == nil {
		tx = DB
	}

	// Don't create one if there is already a PolicyUser for this user
	var pUsers []PolicyUser
	count, err := tx.Where("user_id = ?", u.ID).Count(&pUsers)
	if err != nil {
		msg := fmt.Sprintf("error getting count of policy users for user %s: %s", u.ID, err.Error())
		panic(msg)
	}

	if count > 0 {
		return nil
	}

	// If there is already a Policy with this householdID, create a PolicyUser but not
	//  another Policy
	if householdID != "" {
		gotCreated, err := u.createInitialPolicyUser(tx, householdID)
		if err != nil {
			return err
		}
		if gotCreated {
			return nil
		}
	}

	policy := Policy{
		Name:         u.LastName + " household",
		Type:         api.PolicyTypeHousehold,
		EntityCodeID: HouseholdEntityID(),
	}
	if householdID != "" {
		policy.HouseholdID = nulls.NewString(householdID)
	}

	if err := policy.Create(tx); err != nil {
		return errors.New("unable to create initial policy in CreateInitialPolicy: " + err.Error())
	}

	polUser := PolicyUser{
		PolicyID: policy.ID,
		UserID:   u.ID,
	}

	if err := polUser.Create(tx); err != nil {
		return errors.New("unable to create policy-user in CreateInitialPolicy: " + err.Error())
	}
	return nil
}

func (u *User) createInitialPolicyUser(tx *pop.Connection, householdID string) (bool, error) {
	var policy Policy
	err := tx.Where("household_id = ?", householdID).First(&policy)
	if domain.IsOtherThanNoRows(err) {
		msg := fmt.Sprintf("error fetching policies with household id %s: %s", householdID, err.Error())
		panic(msg)
	}

	//  If it found no policy with the same household_id then we're done.
	//   Otherwise, create a matching PolicyUser
	if policy.ID == uuid.Nil {
		return false, nil
	}

	polUser := PolicyUser{
		PolicyID: policy.ID,
		UserID:   u.ID,
	}

	if err := polUser.Create(tx); err != nil {
		return false, errors.New("unable to create policy-user in CreateInitialPolicy: " + err.Error())
	}
	return true, nil
}

func (u *Users) GetAll(tx *pop.Connection) error {
	return tx.All(u)
}

// OwnsFile returns true if the user owns the file identified by the given ID
func (u *User) OwnsFile(tx *pop.Connection, f File) (bool, error) {
	if u.ID == uuid.Nil {
		return false, errors.New("no user ID provided")
	}

	return u.ID == f.CreatedByID, nil
}

// AttachPhotoFile assigns a previously-stored File to this User as a profile photo
func (u *User) AttachPhotoFile(tx *pop.Connection, fileID uuid.UUID) error {
	var f File
	if err := tx.Find(&f, fileID); err != nil {
		return appErrorFromDB(
			errors.New("error finding file "+err.Error()),
			api.ErrorResourceNotFound,
		)
	}

	isOwner, err := u.OwnsFile(tx, f)
	if err != nil {
		return err
	}

	if !isOwner {
		return api.NewAppError(
			errors.New("user is not owner of PhotoFile, ID: "+fileID.String()),
			api.ErrorNotAuthorized,
			api.CategoryForbidden,
		)
	}

	if err := addFile(tx, u, f); err != nil {
		return err
	}

	u.LoadPhotoFile(tx)
	return nil
}

// DeletePhotoFile deletes the user's profile photo file
func (u *User) DeletePhotoFile(tx *pop.Connection) error {
	if !u.PhotoFileID.Valid {
		log.Warningf("user %v has no PhotoFileID to delete", u.ID)
		return nil
	}

	var f File
	if err := tx.Find(&f, u.PhotoFileID.UUID); err != nil {
		return appErrorFromDB(
			errors.New("error finding user's photo file "+err.Error()),
			api.ErrorResourceNotFound,
		)
	}

	if err := storage.RemoveFile(f.ID.String()); err != nil {
		log.Errorf("error removing file from S3, id='%s', %s", f.ID.String(), err)
	}

	if err := tx.Destroy(&f); err != nil {
		return appErrorFromDB(
			fmt.Errorf("file %d destroy error, %s", f.ID, err),
			api.ErrorUpdateFailure,
		)
	}

	u.PhotoFileID = nulls.UUID{}
	return u.Update(tx)
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
		out[i] = uu.ConvertToAPI(tx, false)
	}
	return out
}

func (u *User) ConvertToAPI(tx *pop.Connection, hydrate bool) api.User {
	u.LoadPhotoFile(tx)

	output := api.User{
		ID:            u.ID,
		Email:         u.Email,
		EmailOverride: u.EmailOverride,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Name:          u.Name(),
		AppRole:       string(u.AppRole),
		LastLoginUTC:  u.LastLoginUTC,
		Country:       u.GetLocation().Country,
		PhotoFileID:   convertUUIDToAPI(u.PhotoFileID),
	}

	if hydrate {
		u.LoadPolicies(tx, false)
		output.Policies = u.Policies.ConvertToAPI(tx)
	}

	if u.PhotoFile != nil {
		f := u.PhotoFile.ConvertToAPI(tx)
		output.PhotoFile = &f
	}

	return output
}

func (u *User) GetLocation() Location {
	return Location{
		City:    u.City,
		State:   u.State,
		Country: u.Country,
	}
}
