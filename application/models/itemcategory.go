package models

import (
	"errors"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"

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
	ID                uuid.UUID              `db:"id"`
	Key               string                 `db:"key"`
	RiskCategoryID    uuid.UUID              `db:"risk_category_id"`
	Name              string                 `db:"name" validate:"required"`
	HelpText          string                 `db:"help_text"`
	Status            api.ItemCategoryStatus `db:"status" validate:"itemCategoryStatus"`
	AutoApproveMax    int                    `db:"auto_approve_max" validate:"min=0"`
	MinimumPremium    int                    `db:"minimum_premium" validate:"min=0"`
	MinimumCoverage   int                    `db:"minimum_coverage" validate:"min=0"`
	MinimumDeductible int                    `db:"minimum_deductible" validate:"min=0"`
	RequireMakeModel  bool                   `db:"require_make_model"`
	PremiumFactor     nulls.Float64          `db:"premium_factor"`
	BillingPeriod     int                    `db:"billing_period"`
	LegacyID          nulls.Int              `db:"legacy_id"`
	CreatedAt         time.Time              `db:"created_at"`
	UpdatedAt         time.Time              `db:"updated_at"`

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

func (i *ItemCategory) ConvertToAPI(tx *pop.Connection) api.ItemCategory {
	i.LoadRiskCategory(tx)

	var premiumFactor string
	if i.PremiumFactor.Valid {
		premiumFactor = domain.PercentString(i.PremiumFactor.Float64)
	}
	return api.ItemCategory{
		ID:                i.ID,
		Key:               i.Key,
		Name:              i.Name,
		HelpText:          i.HelpText,
		RiskCategory:      i.RiskCategory.ConvertToAPI(),
		RequireMakeModel:  i.RequireMakeModel,
		BillingPeriod:     i.GetBillingPeriod(),
		PremiumFactor:     premiumFactor,
		MinimumDeductible: i.MinimumDeductible,
		MinimumPremium:    i.MinimumPremium,
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
	}
}

func (i *ItemCategory) LoadRiskCategory(tx *pop.Connection) {
	if err := tx.Load(i, "RiskCategory"); err != nil {
		panic("database error loading ItemCategory.RiskCategory, " + err.Error())
	}
}

func (i *ItemCategory) GetBillingPeriod() int {
	b := i.BillingPeriod
	if b != domain.BillingPeriodMonthly && b != domain.BillingPeriodAnnual {
		log.Fatalf("invalid billing period found in item category %s", i.Name)
	}
	return b
}

func (i *ItemCategories) ConvertToAPI(tx *pop.Connection) api.ItemCategories {
	cats := make(api.ItemCategories, len(*i))
	for j, ii := range *i {
		cats[j] = ii.ConvertToAPI(tx)
	}
	return cats
}

func (i *ItemCategories) AllEnabled(tx *pop.Connection) error {
	if err := tx.Where("status = ? AND name != 'Other'", api.ItemCategoryStatusEnabled).
		Order("name asc").All(i); err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}

	// "Other" should be the last one in the list
	var other ItemCategory
	if err := tx.Where("name = 'Other'").First(&other); err == nil {
		*i = append(*i, other)
	}

	return nil
}
