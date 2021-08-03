package models

import (
	"time"

	"github.com/gobuffalo/validate/v3"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
)

type PolicyUsers []PolicyUser

type PolicyUser struct {
	ID       uuid.UUID `db:"id"`
	PolicyID uuid.UUID `db:"policy_id"`
	UserID   uuid.UUID `db:"user_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
//  It first adds a UUID to the user if its UUID is empty
func (p *PolicyUser) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

func (p *PolicyUser) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyUser) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}
