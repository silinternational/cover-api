package api

import (
	"fmt"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
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
	Status ClaimStatus `json:"status"`

	// is item repairable?
	IsRepairable *bool `json:"is_repairable"`

	// repair estimate (0.01 USD)
	RepairEstimate Currency `json:"repair_estimate,omitempty"`

	// actual repair cost (0.01 USD)
	RepairActual Currency `json:"repair_actual,omitempty"`

	// replacement estimate (0.01 USD)
	ReplaceEstimate Currency `json:"replace_estimate,omitempty"`

	// actual replacement cost (0.01 USD)
	ReplaceActual Currency `json:"replace_actual,omitempty"`

	// payout option
	PayoutOption PayoutOption `json:"payout_option,omitempty"`

	// payout amount (0.01 USD)
	PayoutAmount Currency `json:"payout_amount,omitempty"`

	// coverage amount at the time the Claim was created (0.01 USD)
	CoverageAmount Currency `json:"coverage_amount,omitempty"`

	// fair market value (0.01 USD)
	FMV Currency `json:"fmv,omitempty"`

	// review date
	//
	// swagger:strfmt date-time
	ReviewDate nulls.Time `json:"review_date,omitempty"`

	// reviewer User ID
	//
	// swagger:strfmt uuid4
	ReviewerID nulls.UUID `json:"reviewer_id,omitempty"`

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
	IsRepairable *bool `json:"is_repairable"`

	// repair estimate (0.01 USD)
	RepairEstimate Currency `json:"repair_estimate"`

	// actual repair cost (0.01 USD)
	RepairActual Currency `json:"repair_actual"`

	// replacement estimate (0.01 USD)
	ReplaceEstimate Currency `json:"replace_estimate"`

	// actual replacement cost (0.01 USD)
	ReplaceActual Currency `json:"replace_actual"`

	// payout option
	PayoutOption PayoutOption `json:"payout_option"`

	// fair market value (0.01 USD)
	FMV Currency `json:"fmv"`
}

// swagger:model
type ClaimItemUpdateInput struct {
	// is item repairable?
	IsRepairable *bool `json:"is_repairable"`

	// repair estimate (0.01 USD)
	RepairEstimate Currency `json:"repair_estimate"`

	// actual repair cost (0.01 USD)
	RepairActual Currency `json:"repair_actual"`

	// replacement estimate (0.01 USD)
	ReplaceEstimate Currency `json:"replace_estimate"`

	// actual replacement cost (0.01 USD)
	ReplaceActual Currency `json:"replace_actual"`

	// payout option
	PayoutOption PayoutOption `json:"payout_option"`

	// fair market value (0.01 USD)
	FMV Currency `json:"fmv"`
}

// GetPayoutOptionDescription provides a user-facing description for the given PayoutOption and minimum deductible
func GetPayoutOptionDescription(option PayoutOption, minimumDeductible Currency, deductibleRate float64) string {
	minString := ""
	if minimumDeductible > 0 {
		minString = fmt.Sprintf(", subject to a minimum deductible of $%s", minimumDeductible)
	}

	rate := domain.PercentString(deductibleRate)

	switch option {
	case PayoutOptionRepair:
		return fmt.Sprintf("Payout is the item's covered value, the repair cost, or %s of the item's fair market value, whichever is less, minus a %s deductible%s.",
			domain.Env.RepairThresholdString, rate, minString)
	case PayoutOptionReplacement:
		return fmt.Sprintf("Payout is the item's covered value or the replacement cost, whichever is less, minus a %s deductible%s.",
			rate, minString)
	case PayoutOptionFMV:
		return fmt.Sprintf("Payout is the item's fair market value minus a %s deductible%s.",
			rate, minString)
	case PayoutOptionFixedFraction:
		return "Payout is a fixed portion of the item's covered value."
	default:
		return ""
	}
}
