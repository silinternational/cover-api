package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// ItemCoverageStatus
//
// may be one of: Draft, Pending, Revision, Approved, Denied, Inactive
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

	// year
	Year string `json:"year"`

	// serial number
	SerialNumber string `json:"serial_number"`

	// coverage amount (0.01 USD)
	CoverageAmount int `json:"coverage_amount"`

	// coverage status
	CoverageStatus ItemCoverageStatus `json:"coverage_status"`

	// how the status changed most recently (for the stewards dashboard)
	StatusChange string `json:"status_change"`

	// message from a reviewer detailing the revisions needed
	StatusReason string `json:"status_reason"`

	// date (yyyy-mm-dd) of item's coverage start date
	CoverageStartDate string `json:"coverage_start_date"`

	// date (yyyy-mm-dd) of item's coverage end date
	//
	// swagger:strfmt date-time
	CoverageEndDate *string `json:"coverage_end_date"`

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

	// billing period, expressed as a number of months
	BillingPeriod int `json:"billing_period"`

	// annual premium (0.01 USD)
	AnnualPremium Currency `json:"annual_premium"`

	// estimated annual premium prorated from now to the end of the year (0.01 USD)
	ProratedAnnualPremium Currency `json:"prorated_annual_premium"`

	// monthly premium (0.01 USD)
	MonthlyPremium *Currency `json:"monthly_premium"`

	// Accountable person assigned to the policy item
	AccountablePerson AccountablePerson `json:"accountable_person"`

	// Can the item be deleted? Set to false if the item is in a state that prevents it from being deleted.
	CanBeDeleted bool `json:"can_be_deleted"`

	// Can the item be updated? Set to false if there is an active claim for the item.
	CanBeUpdated bool `json:"can_be_updated"`
}

// swagger:model
type RecentItems []RecentItem

// swagger:model
type RecentItem struct {
	// The time the item had its coverage_status changed
	// swagger:strfmt date-time
	StatusUpdatedAt time.Time

	Item Item
}

// ItemCreate represents payload for adding an item
// swagger:model
type ItemCreate struct {
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
	RiskCategoryID *uuid.UUID `json:"risk_category_id"`

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

	// coverage status
	CoverageStatus ItemCoverageStatus `json:"coverage_status"`

	// date (yyyy-mm-dd) of item's coverage start date
	CoverageStartDate string `json:"coverage_start_date"`

	// date (yyyy-mm-dd) of item's coverage end date, optional
	//
	// swagger:strfmt date-time
	CoverageEndDate *string `json:"coverage_end_date"`

	// Accountable person ID. Can be either a policy dependent ID or a user ID
	//
	// swagger:strfmt uuid4
	AccountablePersonID uuid.UUID `json:"accountable_person_id"`
}

// swagger:model
type ItemStatusInput struct {
	// message from a reviewer detailing the revisions needed or the reason for denial
	StatusReason string `json:"status_reason"`
}

// ItemUpdate represents payload for updating an item
// swagger:model
type ItemUpdate struct {
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
	RiskCategoryID *uuid.UUID `json:"risk_category_id"`

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

	// Accountable person ID. Can be either a policy dependent ID or a user ID
	//
	// swagger:strfmt uuid4
	AccountablePersonID uuid.UUID `json:"accountable_person_id"`
}
