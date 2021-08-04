package api

import (
	"time"

	"github.com/gofrs/uuid"
)

type ItemCategoryStatus string

const (
	ItemCategoryStatusDraft      = ItemCategoryStatus("Draft")
	ItemCategoryStatusEnabled    = ItemCategoryStatus("Enabled")
	ItemCategoryStatusDeprecated = ItemCategoryStatus("Deprecated")
	ItemCategoryStatusDisabled   = ItemCategoryStatus("Disabled")
)

// ItemCategories is a slice of ItemCategory objects
// swagger:model
type ItemCategories []ItemCategory

// ItemCategory is an item's category object
// swagger:model
type ItemCategory struct {
	ID             uuid.UUID          `json:"id"`
	RiskCategoryID uuid.UUID          `json:"risk_category_id"`
	Name           string             `json:"name"`
	HelpText       string             `json:"help_text"`
	Status         ItemCategoryStatus `json:"status"`
	AutoApproveMax int                `json:"auto_approve_max"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`

	RiskCategory RiskCategory `json:"risk_category"`
}
