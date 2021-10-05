package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// ClaimItemStatus
//
// may be one of: Draft, Review1, Review2, Review3, Revision, Receipt, Approved, Paid, Denied
//
// swagger:model
type ClaimItemStatus string

func (s ClaimItemStatus) WasReviewed() bool {
	switch s {
	case ClaimItemStatusDenied, ClaimItemStatusRevision, ClaimItemStatusReceipt,
		ClaimItemStatusApproved, ClaimItemStatusPaid, ClaimItemStatusReview3:
		return true
	}
	return false
}

const (
	ClaimItemStatusDraft    = ClaimItemStatus(ClaimStatusDraft)
	ClaimItemStatusReview1  = ClaimItemStatus(ClaimStatusReview1)
	ClaimItemStatusReview2  = ClaimItemStatus(ClaimStatusReview2)
	ClaimItemStatusReview3  = ClaimItemStatus(ClaimStatusReview3)
	ClaimItemStatusRevision = ClaimItemStatus(ClaimStatusRevision)
	ClaimItemStatusReceipt  = ClaimItemStatus(ClaimStatusReceipt)
	ClaimItemStatusApproved = ClaimItemStatus(ClaimStatusApproved)
	ClaimItemStatusPaid     = ClaimItemStatus(ClaimStatusPaid)
	ClaimItemStatusDenied   = ClaimItemStatus(ClaimStatusDenied)
)

// PayoutOption
//
// may be one of: Repair, Replacement, FMV, FixedFraction
//
// swagger:model
type PayoutOption string

const (
	PayoutOptionRepair        = PayoutOption("Repair")
	PayoutOptionReplacement   = PayoutOption("Replacement")
	PayoutOptionFMV           = PayoutOption("FMV")
	PayoutOptionFixedFraction = PayoutOption("FixedFraction")
)

// swagger:model
type ClaimItems []ClaimItem

// swagger:model
type ClaimItem struct {
	// claim item ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// item ID
	//
	// swagger:strfmt uuid4
	ItemID uuid.UUID `json:"item_id"`

	// item
	Item Item `json:"item"`

	// claim ID
	//
	// swagger:strfmt uuid4
	ClaimID uuid.UUID `json:"claim_id"`

	// claim item status
	Status ClaimItemStatus `json:"status"`

	// is item repairable?
	IsRepairable bool `json:"is_repairable"`

	// repair estimate (0.01 USD)
	RepairEstimate int `json:"repair_estimate,omitempty"`

	// actual repair cost (0.01 USD)
	RepairActual int `json:"repair_actual,omitempty"`

	// replacement estimate (0.01 USD)
	ReplaceEstimate int `json:"replace_estimate,omitempty"`

	// actual replacement cost (0.01 USD)
	ReplaceActual int `json:"replace_actual,omitempty"`

	// payout option
	PayoutOption PayoutOption `json:"payout_option,omitempty"`

	// payout amount (0.01 USD)
	PayoutAmount int `json:"payout_amount,omitempty"`

	// fair market value (0.01 USD)
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

// swagger:model
type ClaimItemCreateInput struct {
	// item ID
	//
	// swagger:strfmt uuid4
	ItemID uuid.UUID `json:"item_id"`

	// is item repairable?
	IsRepairable bool `json:"is_repairable"`

	// repair estimate (0.01 USD)
	RepairEstimate int `json:"repair_estimate"`

	// actual repair cost (0.01 USD)
	RepairActual int `json:"repair_actual"`

	// replacement estimate (0.01 USD)
	ReplaceEstimate int `json:"replace_estimate"`

	// actual replacement cost (0.01 USD)
	ReplaceActual int `json:"replace_actual"`

	// payout option
	PayoutOption PayoutOption `json:"payout_option"`

	// fair market value (0.01 USD)
	FMV int `json:"fmv"`
}

// swagger:model
type ClaimItemUpdateInput struct {
	// is item repairable?
	IsRepairable bool `json:"is_repairable"`

	// repair estimate (0.01 USD)
	RepairEstimate int `json:"repair_estimate"`

	// actual repair cost (0.01 USD)
	RepairActual int `json:"repair_actual"`

	// replacement estimate (0.01 USD)
	ReplaceEstimate int `json:"replace_estimate"`

	// actual replacement cost (0.01 USD)
	ReplaceActual int `json:"replace_actual"`

	// payout option
	PayoutOption PayoutOption `json:"payout_option"`

	// fair market value (0.01 USD)
	FMV int `json:"fmv"`
}
