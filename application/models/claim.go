package models

import (
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
	api.ClaimEventTypeTheft:      {},
	api.ClaimEventTypeImpact:     {},
	api.ClaimEventTypeLightning:  {},
	api.ClaimEventTypeWater:      {},
	api.ClaimEventTypeEvacuation: {},
	api.ClaimEventTypeOther:      {},
}

var ValidClaimStatus = map[api.ClaimStatus]struct{}{
	api.ClaimStatusDraft:    {},
	api.ClaimStatusPending:  {},
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
	CreatedAt        time.Time          `db:"created_at"`
	UpdatedAt        time.Time          `db:"updated_at"`

	Policy     Policy     `belongs_to:"policies" validate:"-"`
	ClaimItems ClaimItems `has_many:"claim_items" validate:"-"`
	Items      Items      `many_to_many:"claim_items" validate:"-"`
	Reviewer   User       `belongs_to:"users" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (c *Claim) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(c), nil
}

// Create stores the Policy data as a new record in the database.
func (c *Claim) Create(tx *pop.Connection) error {
	return create(tx, c)
}

// Update writes the Policy data to an existing database record.
func (c *Claim) Update(tx *pop.Connection) error {
	return update(tx, c)
}

func (c *Claim) GetID() uuid.UUID {
	return c.ID
}

func (c *Claim) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(c, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (c *Claim) IsActorAllowedTo(tx *pop.Connection, user User, perm Permission, sub SubResource, r *http.Request) bool {
	if user.IsAdmin() {
		return true
	}

	var policy Policy
	if err := policy.FindByID(tx, c.PolicyID); err != nil {
		domain.ErrLogger.Printf("failed to load Policy for Claim: %s", err)
		return false
	}

	policy.LoadMembers(tx, false)

	for _, m := range policy.Members {
		if m.ID == user.ID {
			return true
		}
	}

	return false
}

func (c *Claim) LoadItems(tx *pop.Connection, reload bool) error {
	if len(c.Items) == 0 || reload {
		return tx.Load(c, "Items")
	}
	return nil
}

func (c *Claim) LoadPolicy(tx *pop.Connection, reload bool) error {
	if c.Policy.ID == uuid.Nil || reload {
		return tx.Load(c, "Policy")
	}
	return nil
}

func (c *Claim) LoadReviewer(tx *pop.Connection, reload bool) error {
	if c.ReviewerID.Valid && (c.Reviewer.ID == uuid.Nil || reload) {
		return tx.Load(c, "Reviewer")
	}
	return nil
}

func ConvertClaim(c Claim) api.Claim {
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
	}
}

func ConvertClaims(cs Claims) api.Claims {
	claims := make(api.Claims, len(cs))
	for i, c := range cs {
		claims[i] = ConvertClaim(c)
	}
	return claims
}
