package models

import (
	"net/http"
	"time"

	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"

	"github.com/gofrs/uuid"
)

type Policies []Policy

type PolicyType string

const (
	PolicyTypeHousehold = PolicyType("Household")
	PolicyTypeOU        = PolicyType("OU")
)

var ValidPolicyTypes = map[PolicyType]struct{}{
	PolicyTypeHousehold: {},
	PolicyTypeOU:        {},
}

type Policy struct {
	ID          uuid.UUID  `db:"id"`
	Type        PolicyType `db:"type" validate:"policyType"`
	HouseholdID string     `db:"household_id"`
	CostCenter  string     `db:"cost_center"`
	Account     string     `db:"account"`
	EntityCode  string     `db:"entity_code"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`

	Members PolicyUsers `db:"many_to_many:"policy_users`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *Policy) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

func (p *Policy) GetID() uuid.UUID {
	return p.ID
}

func (p *Policy) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (p *Policy) IsActorAllowedTo(tx *pop.Connection, user User, perm Permission, sub SubResource, r *http.Request) bool {
	if user.IsAdmin() {
		return true
	}

	if len(p.Members) == 0 {
		if err := p.LoadMembers(tx); err != nil {
			domain.ErrLogger.Printf("failed to load members on policy: %s", err)
			return false
		}
	}

	for _, m := range p.Members {
		if m.ID == user.ID {
			return true
		}
	}

	return false
}

// LoadMembers - a simple wrapper method for loading members on the struct
func (p *Policy) LoadMembers(tx *pop.Connection) error {
	if err := tx.Load(p, "Members"); err != nil {
		return err
	}
	return nil
}
