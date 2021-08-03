package models

import (
	"net/http"
	"time"

	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/validate/v3"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
)

type PolicyUsers []PolicyUser

type PolicyUser struct {
	ID       uuid.UUID `db:"id"`
	PolicyID uuid.UUID `db:"policy_id"`
	UserID   uuid.UUID `db:"user_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *PolicyUser) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

func (p *PolicyUser) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyUser) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (p *PolicyUser) IsActorAllowedTo(tx *pop.Connection, user User, perm Permission, sub SubResource, r *http.Request) bool {
	if user.IsAdmin() {
		return true
	}

	var policy Policy
	if err := policy.FindByID(tx, p.PolicyID); err != nil {
		domain.ErrLogger.Printf("failed to load policy for dependent: %s", err)
		return false
	}

	if err := policy.LoadMembers(tx, false); err != nil {
		domain.ErrLogger.Printf("failed to load members on policy: %s", err)
		return false
	}

	for _, m := range policy.Members {
		if m.ID == user.ID {
			return true
		}
	}

	return false
}
