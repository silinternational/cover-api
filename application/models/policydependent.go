package models

import (
	"net/http"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

var ValidPolicyDependentRelationships = map[api.PolicyDependentRelationship]struct{}{
	api.PolicyDependentRelationshipSpouse: {},
	api.PolicyDependentRelationshipChild:  {},
}

type PolicyDependents []PolicyDependent

type PolicyDependent struct {
	ID             uuid.UUID                       `db:"id"`
	PolicyID       uuid.UUID                       `db:"policy_id"`
	Name           string                          `db:"name" validate:"required"`
	Relationship   api.PolicyDependentRelationship `db:"relationship" validate:"policyDependentRelationship"`
	City           string                          `db:"city"`
	State          string                          `db:"state"`
	Country        string                          `db:"country" validate:"required"`
	ChildBirthYear int                             `db:"child_birth_year" validate:"policyDependentChildBirthYear,required_if=Relationship Child"`

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
func (p *PolicyDependent) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	var policy Policy
	if err := policy.FindByID(tx, p.PolicyID); err != nil {
		domain.ErrLogger.Printf("failed to load policy for dependent: %s", err)
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

func (p *PolicyDependent) ConvertToAPI() api.PolicyDependent {
	return api.PolicyDependent{
		ID:             p.ID,
		Name:           p.Name,
		Relationship:   p.Relationship,
		Country:        p.GetLocation(),
		ChildBirthYear: p.ChildBirthYear,
	}
}

func (p *PolicyDependents) ConvertToAPI() api.PolicyDependents {
	deps := make(api.PolicyDependents, len(*p))
	for i, pp := range *p {
		deps[i] = pp.ConvertToAPI()
	}
	return deps
}

func (p *PolicyDependent) GetLocation() string {
	return location(p.City, p.State, p.Country)
}
