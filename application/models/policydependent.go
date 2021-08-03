package models

import (
	"time"

	"github.com/gobuffalo/validate/v3"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
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
