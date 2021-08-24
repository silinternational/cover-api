package models

import (
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/domain"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
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
	LegacyID       nulls.Int              `db:"legacy_id"`
	CreatedAt      time.Time              `db:"created_at"`
	UpdatedAt      time.Time              `db:"updated_at"`

	RiskCategory RiskCategory `belongs_to:"risk_categories" fk_id:"RiskCategoryID" validate:"-"`
}

func (r *ItemCategory) Create(tx *pop.Connection) error {
	return create(tx, r)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (i *ItemCategory) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(i), nil
}

func (i *ItemCategory) GetID() uuid.UUID {
	return i.ID
}

// Create stores the data as a new record in the database.
func (i *ItemCategory) Create(tx *pop.Connection) error {
	return create(tx, i)
}

func (i *ItemCategory) FindByID(tx *pop.Connection, id uuid.UUID) error {
	if err := tx.Find(i, id); err != nil {
		appErr := api.AppError{
			Err:      err,
			Key:      api.ErrorQueryFailure,
			Category: api.CategoryInternal,
		}
		if !domain.IsOtherThanNoRows(err) {
			appErr.Category = api.CategoryUser
		}
		return &appErr
	}
	return nil
}

func ConvertItemCategory(tx *pop.Connection, ic ItemCategory) api.ItemCategory {
	return api.ItemCategory{
		ID:             ic.ID,
		Name:           ic.Name,
		HelpText:       ic.HelpText,
		Status:         ic.Status,
		AutoApproveMax: ic.AutoApproveMax,
		CreatedAt:      ic.CreatedAt,
		UpdatedAt:      ic.UpdatedAt,
	}
}

func ConvertItemCategories(tx *pop.Connection, ics ItemCategories) api.ItemCategories {
	cats := make(api.ItemCategories, len(ics))
	for i, ic := range ics {
		cats[i] = ConvertItemCategory(tx, ic)
	}
	return cats
}
