package models

import (
	"time"

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

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
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
