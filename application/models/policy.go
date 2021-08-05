package models

import (
	"net/http"
	"time"

	"github.com/silinternational/riskman-api/api"

	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"

	"github.com/gofrs/uuid"
)

type Policies []Policy

var ValidPolicyTypes = map[api.PolicyType]struct{}{
	api.PolicyTypeHousehold: {},
	api.PolicyTypeOU:        {},
}

type Policy struct {
	ID          uuid.UUID      `db:"id"`
	Type        api.PolicyType `db:"type" validate:"policyType"`
	HouseholdID string         `db:"household_id" validate:"required_if=Type Household"`
	CostCenter  string         `db:"cost_center" validate:"required_if=Type OU"`
	Account     string         `db:"account" validate:"required_if=Type OU"`
	EntityCode  string         `db:"entity_code" validate:"required_if=Type OU"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`

	Dependents PolicyDependents `has_many:"policy_dependents"`
	Members    Users            `many_to_many:"policy_users"`
	Items      Items            `has_many:"items"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *Policy) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

// Create stores the Policy data as a new record in the database.
func (p *Policy) Create(tx *pop.Connection) error {
	return create(tx, p)
}

// Update writes the Policy data to an existing database record.
func (p *Policy) Update(tx *pop.Connection) error {
	return update(tx, p)
}

func (p *Policy) GetID() uuid.UUID {
	return p.ID
}

func (p *Policy) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (p *Policy) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() || perm == PermissionList {
		return true
	}

	if err := p.LoadMembers(tx, false); err != nil {
		domain.ErrLogger.Printf("failed to load members on policy: %s", err)
		return false
	}

	for _, m := range p.Members {
		if m.ID == actor.ID {
			return true
		}
	}

	return false
}

// LoadMembers - a simple wrapper method for loading members on the struct
func (p *Policy) LoadMembers(tx *pop.Connection, reload bool) error {
	if len(p.Members) == 0 || reload {
		if err := tx.Load(p, "Members"); err != nil {
			return err
		}
	}

	return nil
}

// LoadDependents - a simple wrapper method for loading dependents on the struct
func (p *Policy) LoadDependents(tx *pop.Connection, reload bool) error {
	if len(p.Dependents) == 0 || reload {
		if err := tx.Load(p, "Dependents"); err != nil {
			return err
		}
	}

	return nil
}

// LoadItems - a simple wrapper method for loading items on the struct
func (p *Policy) LoadItems(tx *pop.Connection, reload bool) error {
	if len(p.Items) == 0 || reload {
		if err := tx.Load(p, "Items"); err != nil {
			return err
		}
	}

	return nil
}

func ConvertPolicy(tx *pop.Connection, p Policy) (api.Policy, error) {
	if err := p.LoadMembers(tx, false); err != nil {
		return api.Policy{}, err
	}
	if err := p.LoadDependents(tx, false); err != nil {
		return api.Policy{}, err
	}

	members, err := ConvertPolicyMembers(tx, p.Members)
	if err != nil {
		return api.Policy{}, err
	}

	dependents := ConvertPolicyDependents(tx, p.Dependents)

	return api.Policy{
		ID:          p.ID,
		Type:        p.Type,
		HouseholdID: p.HouseholdID,
		CostCenter:  p.CostCenter,
		Account:     p.Account,
		EntityCode:  p.EntityCode,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Members:     members,
		Dependents:  dependents,
	}, nil
}

func ConvertPolicies(tx *pop.Connection, ps Policies) (api.Policies, error) {
	policies := make(api.Policies, len(ps))
	for i, p := range ps {
		var err error
		policies[i], err = ConvertPolicy(tx, p)
		if err != nil {
			return nil, err
		}
	}

	return policies, nil
}
