package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

// PolicyUserInvite represents an invite for a policy co-manager
type PolicyUserInvite struct {
	ID             uuid.UUID  `db:"id"`
	PolicyID       uuid.UUID  `db:"policy_id" validate:"required"`
	Email          string     `db:"email" validate:"required,email"`
	EmailSentAt    nulls.Time `db:"email_sent_at"`
	EmailSendCount int        `db:"email_send_count" validate:"min=0"`
	InviteeName    string     `db:"invitee_name"`
	InviterName    string     `db:"inviter_name"`
	InviterEmail   string     `db:"inviter_email"`
	InviterMessage string     `db:"inviter_message"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`

	Policy Policy `belongs_to:"policies" validate:"-"`
}

func (i PolicyUserInvite) String() string {
	ji, _ := json.Marshal(i)
	return string(ji)
}

type PolicyUserInvites []PolicyUserInvite

func (i PolicyUserInvites) String() string {
	ji, _ := json.Marshal(i)
	return string(ji)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (i *PolicyUserInvite) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(i), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
// This method is not required and may be deleted.
func (i *PolicyUserInvite) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
// This method is not required and may be deleted.
func (i *PolicyUserInvite) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// Create new invite
// emits domain.EventApiPolicyUserInviteCreated event
func (i *PolicyUserInvite) Create(tx *pop.Connection) error {
	if err := create(tx, i); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiPolicyUserInviteCreated,
		Message: "PolicyUserInvite created",
		Payload: events.Payload{"id": i.ID},
	}
	emitEvent(e)

	return nil
}

func (i *PolicyUserInvite) Update(tx *pop.Connection) error {
	return update(tx, i)
}

func (i *PolicyUserInvite) Save(tx *pop.Connection) error {
	return save(tx, i)
}

func (i *PolicyUserInvite) Destroy(tx *pop.Connection) error {
	if err := tx.Destroy(i); err != nil {
		return appErrorFromDB(fmt.Errorf("error destroying invite: %w", err), api.ErrorQueryFailure)
	}
	return nil
}

func (i *PolicyUserInvite) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(i, id)
}

func (i *PolicyUserInvite) FindByEmailAndPolicyID(tx *pop.Connection, email string, policyID uuid.UUID) error {
	return tx.Where("email = ? and policy_id = ?", email, policyID).First(i)
}

func (i *PolicyUserInvite) GetAcceptURL() string {
	return fmt.Sprintf("%s/invite/%s", domain.Env.UIURL, i.ID)
}

// LoadPolicy - a simple wrapper method for loading the policy
func (i *PolicyUserInvite) LoadPolicy(tx *pop.Connection, reload bool) {
	if i.Policy.ID == uuid.Nil || reload {
		if err := tx.Load(i, "Policy"); err != nil {
			panic("error loading item policy: " + err.Error())
		}
	}
}

func (i *PolicyUserInvite) Accept(tx *pop.Connection, code string, user User) error {
	codeUUID, err := uuid.FromString(code)
	if err != nil {
		err = errors.New("error converting invite code from string to uuid: " + err.Error())
		return api.NewAppError(err, api.ErrorPolicyUserInviteCode, api.CategoryUser)

	}

	if err := i.FindByID(tx, codeUUID); err != nil {
		return appErrorFromDB(fmt.Errorf("failed to load policy invite: %w", err), api.ErrorQueryFailure)
	}

	i.LoadPolicy(tx, true)

	if i.Policy.HouseholdID.Valid && user.StaffID.Valid {
		householdID := GetHHID(user.StaffID.String)
		if householdID != i.Policy.HouseholdID.String {
			return api.NewAppError(err, api.ErrorPolicyUserInviteDifferentHouseholdID, api.CategoryUser)
		}
	}

	policyUser := PolicyUser{
		PolicyID: i.PolicyID,
		UserID:   user.ID,
	}

	if err := policyUser.Create(tx); err != nil {
		return err
	}

	return i.Destroy(tx)
}

// DestroyIfExpired returns nil if the invite is not too old or already accepted.
// Otherwise, it attempts to destroy it and returns an error.
func (i *PolicyUserInvite) DestroyIfExpired(tx *pop.Connection) error {
	now := time.Now().UTC()
	cutoff := now.Add(time.Duration(-domain.Env.InviteLifetimeDays) * domain.DurationDay)
	if i.CreatedAt.After(cutoff) {
		return nil
	}

	if err := i.Destroy(tx); err != nil {
		return fmt.Errorf("error destroying expired invite %w", err)
	}

	return api.NewAppError(
		errors.New("attempt to use expired invite, ID: "+i.ID.String()),
		api.ErrorInviteExpired,
		api.CategoryForbidden,
	)
}

func (i *PolicyUserInvite) ConvertToAPI() api.PolicyUserInvite {

	return api.PolicyUserInvite{
		Email:       i.Email,
		Name:        i.InviteeName,
		EmailSentAt: convertTimeToAPI(i.EmailSentAt),
	}
}

func (i *PolicyUserInvites) ConvertToAPI() api.PolicyUserInvites {
	invites := make(api.PolicyUserInvites, len(*i))
	for i, ii := range *i {
		invites[i] = ii.ConvertToAPI()
	}
	return invites
}
