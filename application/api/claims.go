package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

// ClaimEventType
//
// may be one of: Theft, Impact, Electrical Surge, Water Damage, Evacuation, Other
//
// swagger:model
type ClaimEventType string

// ClaimStatus
//
// may be one of: Draft, Pending, Approved, Denied
//
// swagger:model
type ClaimStatus string

const (
	ClaimEventTypeTheft           = ClaimEventType("Theft")
	ClaimEventTypeImpact          = ClaimEventType("Impact")
	ClaimEventTypeElectricalSurge = ClaimEventType("Electrical Surge")
	ClaimEventTypeWaterDamage     = ClaimEventType("Water Damage")
	ClaimEventTypeEvacuation      = ClaimEventType("Evacuation")
	ClaimEventTypeOther           = ClaimEventType("Other")
)

// swagger:model
type ClaimEventTypeStruct struct {
	Name         ClaimEventType `json:"name"`
	IsRepairable bool           `json:"is_repairable"`
}

var AllClaimEventTypes = []ClaimEventTypeStruct{
	{ClaimEventTypeTheft, false},
	{ClaimEventTypeImpact, true},
	{ClaimEventTypeElectricalSurge, true},
	{ClaimEventTypeWaterDamage, true},
	{ClaimEventTypeEvacuation, false},
	{ClaimEventTypeOther, true},
}

const (
	ClaimStatusDraft    = ClaimStatus("Draft")
	ClaimStatusReview1  = ClaimStatus("Review1")
	ClaimStatusReview2  = ClaimStatus("Review2")
	ClaimStatusReview3  = ClaimStatus("Review3")
	ClaimStatusRevision = ClaimStatus("Revision")
	ClaimStatusReceipt  = ClaimStatus("Receipt")
	ClaimStatusApproved = ClaimStatus("Approved")
	ClaimStatusPaid     = ClaimStatus("Paid")
	ClaimStatusDenied   = ClaimStatus("Denied")
	ClaimStatusInactive = ClaimStatus("Inactive")
)

// swagger:model
type Claims []Claim

// swagger:model
type Claim struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// policy ID
	//
	// swagger:strfmt uuid4
	PolicyID uuid.UUID `json:"policy_id"`

	// reference number
	//
	// human friendly six character string
	// example: AB4331
	ReferenceNumber string `json:"reference_number"`

	// event date
	//
	// swagger:strfmt date-time
	EventDate time.Time `json:"event_date"`

	// event type
	EventType ClaimEventType `json:"event_type"`

	// event description .
	EventDescription string `json:"event_description"`

	// event status
	Status ClaimStatus `json:"status"`

	// review date
	//
	// swagger:strfmt date-time
	ReviewDate nulls.Time `json:"review_date,omitempty"`

	// reviewer ID
	//
	// swagger:strfmt uuid4
	ReviewerID nulls.UUID `json:"reviewer_id,omitempty"`

	// payment date
	//
	// swagger:strfmt date-time
	PaymentDate nulls.Time `json:"payment_date,omitempty"`

	// total payout
	TotalPayout int `json:"total_payout,omitempty"`

	// list of items included in claim
	Items ClaimItems `json:"claim_items"`
}

// swagger:model
type ClaimCreateInput struct {
	// event date
	EventDate time.Time `json:"event_date"`

	EventType ClaimEventType `json:"event_type"`

	// event description
	EventDescription string `json:"event_description"`
}

// swagger:model
type ClaimUpdateInput struct {
	// event date
	EventDate time.Time `json:"event_date"`

	EventType ClaimEventType `json:"event_type"`

	// event description
	EventDescription string `json:"event_description"`
}
