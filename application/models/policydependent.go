package models

import (
	"net/http"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"
)

type PolicyDependents []PolicyDependent

type PolicyDependent struct {
	ID        uuid.UUID `db:"id"`
	PolicyID  uuid.UUID `db:"policy_id"`
	Name      string    `db:"name" validate:"required"`
	BirthYear int       `db:"birth_year" validate:"required"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *PolicyDependent) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

func (p *PolicyDependent) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyDependent) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}

func (p *PolicyDependent) Create(tx *pop.Connection) error {
	return create(tx, p)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (p *PolicyDependent) IsActorAllowedTo(tx *pop.Connection, user User, perm Permission, sub SubResource, r *http.Request) bool {
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

func ConvertPolicyDependent(tx *pop.Connection, d PolicyDependent) api.PolicyDependent {
	return api.PolicyDependent{
		ID:        d.ID,
		Name:      d.Name,
		BirthYear: d.BirthYear,
	}
}

func ConvertPolicyDependents(tx *pop.Connection, ds PolicyDependents) api.PolicyDependents {
	deps := make(api.PolicyDependents, len(ds))
	for i, d := range ds {
		deps[i] = ConvertPolicyDependent(tx, d)
	}
	return deps
}
