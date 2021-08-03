package models

import (
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// RiskCategories is a slice of RiskCategory objects
type RiskCategories []RiskCategory

// RiskCategory model
type RiskCategory struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name" validate:"required"`
	PolicyMax int       `db:"policy_max" validate:"required"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (r *RiskCategory) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(r), nil
}

func (r *RiskCategory) GetID() uuid.UUID {
	return r.ID
}

func (r *RiskCategory) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(r, id)
}
