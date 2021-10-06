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

	RequireMakeModel bool `json:"require_make_model"`
}
