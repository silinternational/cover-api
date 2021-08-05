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

type ItemCategories []ItemCategory

type ItemCategory struct {
	ID             uuid.UUID          `json:"id"`
	Name           string             `json:"name"`
	HelpText       string             `json:"help_text"`
	Status         ItemCategoryStatus `json:"status"`
	AutoApproveMax int                `json:"auto_approve_max"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}
