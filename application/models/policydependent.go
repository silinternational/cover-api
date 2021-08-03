package models

import (
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
)

type PolicyDependents []PolicyDependent

type PolicyDependent struct {
	ID        uuid.UUID `json:"-" db:"id"`
	PolicyID  uuid.UUID `json:"policy_id" db:"policy_id"`
	Name      string    `json:"name" db:"name"`
	BirthYear int       `json:"birth_year" db:"birth_year"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (p *PolicyDependent) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyDependent) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}
