package models

import (
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// ItemCategories is a slice of ItemCategory objects
type ItemCategories []ItemCategory

// ItemCategory model
type ItemCategory struct {
	ID             uuid.UUID `db:"id"`
	RiskCategoryID uuid.UUID `db:"risk_category_id"`
	Name           string    `db:"name" validate:"required"`
	HelpText       string    `db:"help_text"`
	Status         string    `db:"status" validate:"required"`
	AutoApproveMax int       `db:"auto_approve_max"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`

	RiskCategory RiskCategory `belongs_to:"risk_categories" fk_id:"RiskCategoryID" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (r *ItemCategory) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(r), nil
}

func (r *ItemCategory) GetID() uuid.UUID {
	return r.ID
}

func (r *ItemCategory) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(r, id)
}
