package models

import (
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
)

type PolicyUsers []PolicyUser

type PolicyUser struct {
	ID       uuid.UUID `json:"-" db:"id"`
	PolicyID uuid.UUID `json:"policy_id" db:"policy_id"`
	UserID   uuid.UUID `json:"user_id" db:"user_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (p *PolicyUser) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyUser) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}
