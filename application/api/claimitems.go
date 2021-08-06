package api

import (
	"time"

	"github.com/gofrs/uuid"
)

type ClaimItemStatus string

const (
	ClaimItemStatusPending  = ClaimItemStatus("Pending")
	ClaimItemStatusApproved = ClaimItemStatus("Approved")
	ClaimItemStatusDenied   = ClaimItemStatus("Denied")
)

type ClaimItems []ClaimItem

type ClaimItem struct {
	ID              uuid.UUID       `json:"id"`
	ClaimID         uuid.UUID       `json:"claim_id"`
	ItemID          uuid.UUID       `json:"item_id"`
	Status          ClaimItemStatus `json:"status"`
	IsRepairable    bool            `json:"is_repairable"`
	RepairEstimate  int             `json:"repair_estimate,omitempty"`
	RepairActual    int             `json:"repair_actual,omitempty"`
	ReplaceEstimate int             `json:"replace_estimate,omitempty"`
	ReplaceActual   int             `json:"replace_actual,omitempty"`
	PayoutOption    string          `json:"payout_option,omitempty"`
	PayoutAmount    int             `json:"payout_amount,omitempty"`
	FMV             int             `json:"fmv,omitempty"`
	ReviewDate      time.Time       `json:"review_date,omitempty"`
	ReviewerID      uuid.UUID       `json:"reviewer_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
