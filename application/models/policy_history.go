package models

import (
	"fmt"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
)

type PolicyHistories []PolicyHistory

type PolicyHistory struct {
	ID          uuid.UUID  `db:"id"`
	PolicyID    uuid.UUID  `db:"policy_id"`
	UserID      uuid.UUID  `db:"user_id"`
	Action      string     `db:"action"`
	FieldName   string     `db:"field_name"`
	ItemID      nulls.UUID `db:"item_id"`
	Description string     `db:"description"`
	OldValue    string     `db:"old_value"`
	NewValue    string     `db:"new_value"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *PolicyHistory) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

func (p *PolicyHistory) Create(tx *pop.Connection) error {
	return create(tx, p)
}

func (p *PolicyHistory) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyHistory) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}

func (p *PolicyHistory) GenerateDescription(tx *pop.Connection) PolicyHistory {
	var user User
	if err := user.FindByID(tx, p.UserID); err != nil {
		panic("failed to find user by ID " + err.Error())
	}

	switch p.Action {
	case api.HistoryActionCreate:
		p.Description = fmt.Sprintf("record created by %s", user.Name())
	case api.HistoryActionUpdate:
		p.Description = fmt.Sprintf(`field %s changed from "%s" to "%s" by %s`,
			p.FieldName, p.OldValue, p.NewValue, user.Name())
	default:
		p.Description = p.Action
	}
	return *p
}
