package api

import (
	"time"

	"github.com/gofrs/uuid"
)

type ItemCoverageStatus string

const (
	ItemCoverageStatusDraft    = ItemCoverageStatus("Draft")
	ItemCoverageStatusPending  = ItemCoverageStatus("Pending")
	ItemCoverageStatusApproved = ItemCoverageStatus("Approved")
	ItemCoverageStatusDenied   = ItemCoverageStatus("Denied")
)

// swagger:model
type Items []Item

// Item represents a single item on a policy
// swagger:model
type Item struct {
	// unique id (uuid) for thread
	//
	// swagger:strfmt uuid4
	// unique: true
	// example: 63d5b060-1460-4348-bdf0-ad03c105a8d5
	ID uuid.UUID `json:"id"`

	Name              string             `json:"name"`
	CategoryID        uuid.UUID          `json:"category_id"`
	InStorage         bool               `json:"in_storage"`
	Country           string             `json:"country"`
	Description       string             `json:"description"`
	Make              string             `json:"make"`
	Model             string             `json:"model"`
	SerialNumber      string             `json:"serial_number"`
	CoverageAmount    int                `json:"coverage_amount"`
	PurchaseDate      time.Time          `json:"purchase_date"`
	CoverageStatus    ItemCoverageStatus `json:"coverage_status"`
	CoverageStartDate time.Time          `json:"coverage_start_date"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`

	Category ItemCategory `json:"category"`
}
