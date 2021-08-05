package models

import (
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

type PolicyHistory struct {
	ID          uuid.UUID `db:"id"`
	PolicyID    uuid.UUID `db:"policy_id"`
	UserID      uuid.UUID `db:"user_id"`
	Action      string    `db:"action"`
	ItemID      uuid.UUID `db:"item_id"`
	Description string    `db:"description"`
	OldValue    string    `db:"old_value"`
	NewValue    string    `db:"new_value"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *PolicyHistory) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

func (p *PolicyHistory) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyHistory) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}
