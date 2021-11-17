package models

import (
	"time"

	"github.com/silinternational/cover-api/api"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

const (
	RiskCategoryMobileIDString     = "3be38915-7092-44f2-90ef-26f48214b34f"
	RiskCategoryStationaryIDString = "7bed3c00-23cf-4282-b2b8-da89426cef2f"
)

var (
	RiskCategoryMobileID     = uuid.FromStringOrNil(RiskCategoryMobileIDString)
	RiskCategoryStationaryID = uuid.FromStringOrNil(RiskCategoryStationaryIDString)
)

// RiskCategories is a slice of RiskCategory objects
type RiskCategories []RiskCategory

// RiskCategory model
type RiskCategory struct {
	ID         uuid.UUID `db:"id"`
	Name       string    `db:"name" validate:"required"`
	PolicyMax  int       `db:"policy_max" validate:"required,min=0"`
	CostCenter string    `db:"cost_center" validate:"required"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (r *RiskCategory) Create(tx *pop.Connection) error {
	return create(tx, r)
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

func (r *RiskCategory) ConvertToAPI() api.RiskCategory {
	return api.RiskCategory{
		ID:        r.ID,
		Name:      r.Name,
		PolicyMax: r.PolicyMax,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
