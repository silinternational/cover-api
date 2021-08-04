package models

import (
	"time"

	"github.com/silinternational/riskman-api/api"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

var ValidItemCategoryStatuses = map[api.ItemCategoryStatus]struct{}{
	api.ItemCategoryStatusDraft:      {},
	api.ItemCategoryStatusEnabled:    {},
	api.ItemCategoryStatusDeprecated: {},
	api.ItemCategoryStatusDisabled:   {},
}

// ItemCategories is a slice of ItemCategory objects
type ItemCategories []ItemCategory

// ItemCategory model
type ItemCategory struct {
	ID             uuid.UUID              `db:"id"`
	RiskCategoryID uuid.UUID              `db:"risk_category_id"`
	Name           string                 `db:"name" validate:"required"`
	HelpText       string                 `db:"help_text"`
	Status         api.ItemCategoryStatus `db:"status" validate:"itemCategoryStatus"`
	AutoApproveMax int                    `db:"auto_approve_max"`
	CreatedAt      time.Time              `db:"created_at"`
	UpdatedAt      time.Time              `db:"updated_at"`

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

// LoadCategory - a simple wrapper method for loading an item category on the struct
func (r *ItemCategory) LoadRiskCategory(tx *pop.Connection, reload bool) error {
	if r.RiskCategory.ID == uuid.Nil || reload {
		if err := tx.Load(r, "RiskCategory"); err != nil {
			return err
		}
	}

	return nil
}

func ConvertItemCategory(tx *pop.Connection, iCat ItemCategory) (api.ItemCategory, error) {
	if err := iCat.LoadRiskCategory(tx, false); err != nil {
		return api.ItemCategory{}, err
	}

	rCat, err := ConvertRiskCategory(iCat.RiskCategory)
	if err != nil {
		return api.ItemCategory{}, err
	}
	return api.ItemCategory{
		ID:             iCat.ID,
		Name:           iCat.Name,
		HelpText:       iCat.HelpText,
		RiskCategory:   rCat,
		Status:         iCat.Status,
		AutoApproveMax: iCat.AutoApproveMax,
		CreatedAt:      iCat.CreatedAt,
		UpdatedAt:      iCat.UpdatedAt,
	}, nil
}

func ConvertItemCategories(tx *pop.Connection, iCats ItemCategories) (api.ItemCategories, error) {
	apiICs := make(api.ItemCategories, len(iCats))
	for i, c := range iCats {
		var err error
		apiICs[i], err = ConvertItemCategory(tx, c)
		if err != nil {
			return nil, err
		}
	}

	return apiICs, nil
}
