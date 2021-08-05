package models

import (
	"net/http"
	"time"

	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/validate/v3"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
)

type PolicyDependentRelationship string

const (
	PolicyDependentRelationshipSpouse = PolicyDependentRelationship("Spouse")
	PolicyDependentRelationshipChild  = PolicyDependentRelationship("Child")

	MaximumChildAge = 26
)

var (
	ValidPolicyDependentRelationships = map[PolicyDependentRelationship]struct{}{
		PolicyDependentRelationshipSpouse: {},
		PolicyDependentRelationshipChild:  {},
	}

	MinimumChildBirthYear = time.Now().UTC().Year() - MaximumChildAge
)

type PolicyDependents []PolicyDependent

type PolicyDependent struct {
	ID             uuid.UUID                   `db:"id"`
	PolicyID       uuid.UUID                   `db:"policy_id"`
	Name           string                      `db:"name" validate:"required"`
	Relationship   PolicyDependentRelationship `db:"relationship" validate:"validatePolicyDependentRelationship"`
	Location       string                      `db:"location" validate:"required"`
	ChildBirthYear int                         `db:"child_birth_year" validate:"policyDependentChildBirthYear,required_if=Relationship Child"`

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
