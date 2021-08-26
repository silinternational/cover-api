package models

import (
	"errors"
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

func (i *ItemCategory) Update(tx *pop.Connection) error {
	return update(tx, i)
}

func (i *ItemCategory) FindByID(tx *pop.Connection, id uuid.UUID) error {
	if err := tx.Find(i, id); err != nil {
		if domain.IsOtherThanNoRows(err) {
			return api.NewAppError(err, api.ErrorQueryFailure, api.CategoryInternal)
		}
		return api.NewAppError(errors.New("invalid category"), api.ErrorInvalidCategory, api.CategoryUser)
	}
	return nil
}

func ConvertItemCategory(tx *pop.Connection, ic ItemCategory) api.ItemCategory {
	ic.LoadRiskCategory(tx)
	return api.ItemCategory{
		ID:             ic.ID,
		Name:           ic.Name,
		HelpText:       ic.HelpText,
		Status:         ic.Status,
		AutoApproveMax: ic.AutoApproveMax,
		RiskCategory:   ConvertRiskCategory(ic.RiskCategory),
		CreatedAt:      ic.CreatedAt,
		UpdatedAt:      ic.UpdatedAt,
	}
}

func (i *ItemCategory) LoadRiskCategory(tx *pop.Connection) {
	if err := tx.Load(i, "RiskCategory"); err != nil {
		panic("database error loading ItemCategory.RiskCategory, " + err.Error())
	}
}

func ConvertItemCategories(tx *pop.Connection, ics ItemCategories) api.ItemCategories {
	cats := make(api.ItemCategories, len(ics))
	for i, ic := range ics {
		cats[i] = ConvertItemCategory(tx, ic)
	}
	return cats
}
