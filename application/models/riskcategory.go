package models

import (
	"time"

	"github.com/silinternational/riskman-api/api"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

const (
	RiskCategoryMobileIDString     = "3be38915-7092-44f2-90ef-26f48214b34f"
	RiskCategoryStationaryIDString = "7bed3c00-23cf-4282-b2b8-da89426cef2f"
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

func RiskCategoryMobileID() uuid.UUID {
	return uuid.FromStringOrNil(RiskCategoryMobileIDString)
}

func RiskCategoryStationaryID() uuid.UUID {
	return uuid.FromStringOrNil(RiskCategoryStationaryIDString)
}

func ConvertRiskCategory(rCat RiskCategory) api.RiskCategory {
	return api.RiskCategory{
		ID:        rCat.ID,
		Name:      rCat.Name,
		PolicyMax: rCat.PolicyMax,
		CreatedAt: rCat.CreatedAt,
		UpdatedAt: rCat.UpdatedAt,
	}
}

func ConvertRiskCategories(iCats RiskCategories) api.RiskCategories {
	apiICs := make(api.RiskCategories, len(iCats))
	for i, c := range iCats {
		apiICs[i] = ConvertRiskCategory(c)
	}

	return apiICs
}
