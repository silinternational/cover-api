package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"
)

var ValidClaimEventTypes = map[api.ClaimEventType]struct{}{
	api.ClaimEventTypeTheft:           {},
	api.ClaimEventTypeImpact:          {},
	api.ClaimEventTypeElectricalSurge: {},
	api.ClaimEventTypeWaterDamage:     {},
	api.ClaimEventTypeEvacuation:      {},
	api.ClaimEventTypeOther:           {},
}

var ValidClaimStatus = map[api.ClaimStatus]struct{}{
	api.ClaimStatusDraft:    {},
	api.ClaimStatusPending:  {},
	api.ClaimStatusRevision: {},
	api.ClaimStatusApproved: {},
	api.ClaimStatusDenied:   {},
}

type Claims []Claim

type Claim struct {
	ID               uuid.UUID          `db:"id"`
	PolicyID         uuid.UUID          `db:"policy_id" validate:"required"`
	EventDate        time.Time          `db:"event_date" validate:"required_unless=Status Draft"`
	EventType        api.ClaimEventType `db:"event_type" validate:"claimEventType,required_unless=Status Draft"`
	EventDescription string             `db:"event_description" validate:"required_unless=Status Draft"`
	Status           api.ClaimStatus    `db:"status" validate:"claimStatus"`
	ReviewDate       nulls.Time         `db:"review_date"`
	ReviewerID       nulls.UUID         `db:"reviewer_id"`
	PaymentDate      nulls.Time         `db:"payment_date"`
	TotalPayout      int                `db:"total_payout"`
	LegacyID         nulls.Int          `db:"legacy_id"`
	CreatedAt        time.Time          `db:"created_at"`
	UpdatedAt        time.Time          `db:"updated_at"`

	Policy     Policy     `belongs_to:"policies" validate:"-"`
	ClaimItems ClaimItems `has_many:"claim_items" validate:"-"`
	Reviewer   User       `belongs_to:"users" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (c *Claim) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(c), nil
}

// Create stores the Claim data as a new record in the database.
func (c *Claim) Create(tx *pop.Connection) error {
	return create(tx, c)
}

// Update writes the Claim data to an existing database record.
func (c *Claim) Update(tx *pop.Connection, oldStatus api.ClaimStatus) error {
	validTrans, err := isClaimTransitionValid(oldStatus, c.Status)
	if err != nil {
		panic(err)
	}
	if !validTrans {
		err := fmt.Errorf("invalid claim status transition from %s to %s",
			oldStatus, c.Status)
		appErr := api.NewAppError(err, api.ErrorValidation, api.CategoryUser)
		return appErr
	}
	return update(tx, c)
}

func (c *Claim) GetID() uuid.UUID {
	return c.ID
}

func (c *Claim) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(c, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (c *Claim) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	if perm == PermissionList || perm == PermissionCreate {
		return true
	}

	var policy Policy
	if err := policy.FindByID(tx, c.PolicyID); err != nil {
		domain.ErrLogger.Printf("failed to load Policy %s for Claim: %s", c.PolicyID, err)
		return false
	}

	policy.LoadMembers(tx, false)

	for _, m := range policy.Members {
		if m.ID == actor.ID {
			return true
		}
	}

	return false
}

func claimStatusTransitions() map[api.ClaimStatus][]api.ClaimStatus {
	return map[api.ClaimStatus][]api.ClaimStatus{
		api.ClaimStatusDraft: {
			api.ClaimStatusPending,
		},
		api.ClaimStatusPending: {
			api.ClaimStatusRevision,
			api.ClaimStatusApproved,
			api.ClaimStatusDenied,
		},
		api.ClaimStatusRevision: {
			api.ClaimStatusPending,
		},
		api.ClaimStatusApproved: {},
		api.ClaimStatusDenied:   {},
	}
}

func isClaimTransitionValid(status1, status2 api.ClaimStatus) (bool, error) {
	if status1 == status2 {
		return true, nil
	}
	targets, ok := claimStatusTransitions()[status1]
	if !ok {
		return false, errors.New("unexpected initial status - " + string(status1))
	}

	for _, target := range targets {
		if status2 == target {
			return true, nil
		}
	}

	return false, nil
}

func (c *Claim) LoadClaimItems(tx *pop.Connection, reload bool) {
	if len(c.ClaimItems) == 0 || reload {
		if err := tx.Load(c, "ClaimItems"); err != nil {
			panic("database error loading Claim.ClaimItems, " + err.Error())
		}
	}
}

func (c *Claim) LoadPolicy(tx *pop.Connection, reload bool) {
	if c.Policy.ID == uuid.Nil || reload {
		if err := tx.Load(c, "Policy"); err != nil {
			panic("database error loading Claim.Policy, " + err.Error())
		}
	}
}

func (c *Claim) LoadReviewer(tx *pop.Connection, reload bool) {
	if c.ReviewerID.Valid && (c.Reviewer.ID == uuid.Nil || reload) {
		if err := tx.Load(c, "Reviewer"); err != nil {
			panic("database error loading Claim.Reviewer, " + err.Error())
		}
	}
}

func ConvertClaim(tx *pop.Connection, c Claim) api.Claim {
	c.LoadClaimItems(tx, true)

	return api.Claim{
		ID:               c.ID,
		PolicyID:         c.PolicyID,
		EventDate:        c.EventDate,
		EventType:        c.EventType,
		EventDescription: c.EventDescription,
		Status:           c.Status,
		ReviewDate:       c.ReviewDate,
		ReviewerID:       c.ReviewerID,
		PaymentDate:      c.PaymentDate,
		TotalPayout:      c.TotalPayout,
		Items:            ConvertClaimItems(tx, c.ClaimItems),
	}
}

func ConvertClaims(tx *pop.Connection, cs Claims) api.Claims {
	claims := make(api.Claims, len(cs))
	for i, c := range cs {
		claims[i] = ConvertClaim(tx, c)
	}
	return claims
}

func CovertClaimCreateInput(input api.ClaimCreateInput) Claim {
	return Claim{
		EventDate:        input.EventDate,
		EventType:        input.EventType,
		EventDescription: input.EventDescription,
		Status:           api.ClaimStatusDraft,
	}
}
