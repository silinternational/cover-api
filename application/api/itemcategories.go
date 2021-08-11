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

	// name
	Name string `json:"name"`

	// help text
	HelpText string `json:"help_text"`

	// status
	Status ItemCategoryStatus `json:"status"`

	// auto-approve maximum claim amount
	AutoApproveMax int `json:"auto_approve_max"`

	// date-time created
	//
	// swagger:strfmt date-time
	CreatedAt time.Time `json:"created_at"`

	// date-time last updated
	//
	// swagger:strfmt date-time
	UpdatedAt time.Time `json:"updated_at"`
}
