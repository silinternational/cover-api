package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// ItemCategoryStatus
//
// may be one of: Draft, Enabled, Deprecated, Disabled
//
// swagger:model
type ItemCategoryStatus string

const (
	ItemCategoryStatusDraft      = ItemCategoryStatus("Draft")
	ItemCategoryStatusEnabled    = ItemCategoryStatus("Enabled")
	ItemCategoryStatusDeprecated = ItemCategoryStatus("Deprecated")
	ItemCategoryStatusDisabled   = ItemCategoryStatus("Disabled")
)

// swagger:model
type ItemCategories []ItemCategory

// swagger:model
type ItemCategory struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// unique key for indexing icons or other UI data
	Key string `json:"key"`

	// risk category assigned to new items by default -- can be overridden by a user with sufficient permissions
	RiskCategory RiskCategory `json:"risk_category"`

	// name
	Name string `json:"name"`

	// help text
	HelpText string `json:"help_text"`

	// date-time created
	//
	// swagger:strfmt date-time
	CreatedAt time.Time `json:"created_at"`

	// date-time last updated
	//
	// swagger:strfmt date-time
	UpdatedAt time.Time `json:"updated_at"`

	// whether make and model are required in order for item coverage to be auto approved
	RequireMakeModel bool `json:"require_make_model"`

	// billing period, expressed as a number of months
	BillingPeriod int `json:"billing_period"`

	// the premium factor for this category
	PremiumFactor string `json:"premium_factor"`

	// the minimum deductible amount (in units of 0.01 USD)
	MinimumDeductible int `json:"minimum_deductible"`

	// Minimum premium amount. Any premium bill that would be less than this amount will be charged
	// this amount instead. (in units of 0.01 USD)
	MinimumPremium int `json:"minimum_premium"`
}
