package models

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

type ClaimHistories []ClaimHistory

type ClaimHistory struct {
	ID          uuid.UUID  `db:"id"`
	ClaimID     uuid.UUID  `db:"claim_id"`
	ClaimItemID nulls.UUID `db:"claim_item_id"`
	UserID      uuid.UUID  `db:"user_id"`
	Action      string     `db:"action"`
	FieldName   string     `db:"field_name"`
	OldValue    string     `db:"old_value"`
	NewValue    string     `db:"new_value"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (ch *ClaimHistory) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(ch), nil
}

func (ch *ClaimHistory) Create(tx *pop.Connection) error {
	return create(tx, ch)
}

func (ch *ClaimHistory) GetID() uuid.UUID {
	return ch.ID
}

func (ch *ClaimHistory) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(ch, id)
}
