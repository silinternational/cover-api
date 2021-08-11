package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// ClaimItemStatus
//
// may be one of: Pending, Approved, Denied
//
// swagger:model
type ClaimItemStatus string

const (
	ClaimItemStatusPending  = ClaimItemStatus("Pending")
	ClaimItemStatusApproved = ClaimItemStatus("Approved")
	ClaimItemStatusDenied   = ClaimItemStatus("Denied")
)

// swagger:model
type ClaimItems []ClaimItem

// swagger:model
type ClaimItem struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// claim ID
	//
	// swagger:strfmt uuid4
	ClaimID uuid.UUID `json:"claim_id"`

	// item ID
	//
	// swagger:strfmt uuid4
	ItemID uuid.UUID `json:"item_id"`

	// claim item status
	Status ClaimItemStatus `json:"status"`

	// is item repairable?
	IsRepairable bool `json:"is_repairable"`

	// repair estimate (USD)
	RepairEstimate int `json:"repair_estimate,omitempty"`

	// actual repair cost (USD)
	RepairActual int `json:"repair_actual,omitempty"`

	// replacement estimate (USD)
	ReplaceEstimate int `json:"replace_estimate,omitempty"`

	// actual replacement cost (USD)
	ReplaceActual int `json:"replace_actual,omitempty"`

	// payout option
	PayoutOption string `json:"payout_option,omitempty"`

	// payout amount (USD)
	PayoutAmount int `json:"payout_amount,omitempty"`

	// fair market value (USD)
	FMV int `json:"fmv,omitempty"`

	// review date
	//
	// swagger:strfmt date-time
	ReviewDate time.Time `json:"review_date,omitempty"`

	// reviewer User ID
	//
	// swagger:strfmt uuid4
	ReviewerID uuid.UUID `json:"reviewer_id,omitempty"`

	// date-time created
	//
	// swagger:strfmt date-time
	CreatedAt time.Time `json:"created_at"`

	// date-time last updated
	//
	// swagger:strfmt date-time
	UpdatedAt time.Time `json:"updated_at"`
}
