package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

// ClaimEventType
//
// may be one of: Theft, Impact, Electrical, Water, Evacuation, Other
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
	ClaimEventTypeTheft      = ClaimEventType("Theft")
	ClaimEventTypeImpact     = ClaimEventType("Impact")
	ClaimEventTypeElectrical = ClaimEventType("Electrical")
	ClaimEventTypeWater      = ClaimEventType("Water")
	ClaimEventTypeEvacuation = ClaimEventType("Evacuation")
	ClaimEventTypeOther      = ClaimEventType("Other")

	ClaimStatusDraft    = ClaimStatus("Draft")
	ClaimStatusPending  = ClaimStatus("Pending")
	ClaimStatusRevision = ClaimStatus("Revision")
	ClaimStatusApproved = ClaimStatus("Approved")
	ClaimStatusDenied   = ClaimStatus("Denied")
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
	Items ClaimItems `json:"items"`
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
