package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

// ItemCoverageStatus
//
// may be one of: Draft, Pending, Approved, Denied
//
// swagger:model
type ItemCoverageStatus string

const (
	ItemCoverageStatusDraft    = ItemCoverageStatus("Draft")
	ItemCoverageStatusPending  = ItemCoverageStatus("Pending")
	ItemCoverageStatusRevision = ItemCoverageStatus("Revision")
	ItemCoverageStatusApproved = ItemCoverageStatus("Approved")
	ItemCoverageStatusDenied   = ItemCoverageStatus("Denied")
	ItemCoverageStatusInactive = ItemCoverageStatus("Inactive")
)

// swagger:model
type Items []Item

// Item represents a single item on a policy
// swagger:model
type Item struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	Name string `json:"name"`

	// is item in storage?
	InStorage bool `json:"in_storage"`

	// country where item is located
	Country string `json:"country"`

	// item description
	Description string `json:"description"`

	// policy ID
	//
	// swagger:strfmt uuid4
	PolicyID uuid.UUID `json:"policy_id"`

	// make (manufacturer)
	Make string `json:"make"`

	// model
	Model string `json:"model"`

	// serial number
	SerialNumber string `json:"serial_number"`

	// coverage amount (0.01 USD)
	CoverageAmount int `json:"coverage_amount"`

	// date (yyyy-mm-dd) of item's purchase
	PurchaseDate string `json:"purchase_date"`

	// coverage status
	CoverageStatus ItemCoverageStatus `json:"coverage_status"`

	// date (yyyy-mm-dd) of item's coverage start date
	CoverageStartDate string `json:"coverage_start_date"`

	// The time the item was created
	//
	// swagger:strfmt date-time
	CreatedAt time.Time `json:"created_at"`

	// The time the item was last updated
	//
	// swagger:strfmt date-time
	UpdatedAt time.Time `json:"updated_at"`

	// item category
	Category ItemCategory `json:"category"`

	// risk category
	RiskCategory RiskCategory `json:"risk_category"`

	// annual premium (0.01 USD)
	AnnualPremium int `json:"annual_premium"`

	// Name of accountable person
	AccountablePerson string `json:"accountable_person"`
}

// ItemInput represents payload for adding an item
// swagger:model
type ItemInput struct {
	// name
	Name string `json:"name"`

	// category ID
	//
	// swagger:strfmt uuid4
	CategoryID uuid.UUID `json:"category_id"`

	// risk category ID, should only be set if the user has adequate permissions to override the risk category
	// assigned to the item's category
	//
	// swagger:strfmt uuid4
	RiskCategoryID nulls.UUID `json:"risk_category_id"`

	// is item in storage?
	InStorage bool `json:"in_storage"`

	// country where item is located
	Country string `json:"country"`

	// item description
	Description string `json:"description"`

	// make (manufacturer)
	Make string `json:"make"`

	// model
	Model string `json:"model"`

	// serial number
	SerialNumber string `json:"serial_number"`

	// coverage amount (0.01 USD)
	CoverageAmount int `json:"coverage_amount"`

	// date (yyyy-mm-dd) of item's purchase
	PurchaseDate string `json:"purchase_date"`

	// coverage status
	CoverageStatus ItemCoverageStatus `json:"coverage_status"`

	// date (yyyy-mm-dd) of item's coverage start date
	CoverageStartDate string `json:"coverage_start_date"`

	// Accountable person ID. Can be either a policy dependent ID or a user ID
	AccountablePersonID uuid.UUID `json:"accountable_person_id"`
}
