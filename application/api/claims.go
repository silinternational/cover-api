package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

type (
	ClaimEventType string
	ClaimStatus    string
)

const (
	ClaimEventTypeTheft      = ClaimEventType("Theft")
	ClaimEventTypeImpact     = ClaimEventType("Impact")
	ClaimEventTypeLightning  = ClaimEventType("Lightning")
	ClaimEventTypeWater      = ClaimEventType("Water")
	ClaimEventTypeEvacuation = ClaimEventType("Evacuation")
	ClaimEventTypeOther      = ClaimEventType("Other")

	ClaimStatusDraft    = ClaimStatus("Draft")
	ClaimStatusPending  = ClaimStatus("Pending")
	ClaimStatusApproved = ClaimStatus("Approved")
	ClaimStatusDenied   = ClaimStatus("Denied")
)

// swagger:model
type Claims []Claim

// swagger:model
type Claim struct {
	// unique ID
	//
	// read only: true
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// policy ID
	//
	// swagger:strfmt uuid4
	PolicyID uuid.UUID `json:"policy_id"`

	// event date
	EventDate time.Time `json:"event_date"`

	// event type, one of: Theft, Impact, Lightning, Water, Evacuation, Other
	EventType ClaimEventType `json:"event_type"`

	// event description
	EventDescription string `json:"event_description"`

	// event status
	Status ClaimStatus `json:"status"`

	// review date
	ReviewDate nulls.Time `json:"review_date,omitempty"`

	// reviewer ID
	ReviewerID nulls.UUID `json:"reviewer_id,omitempty"`

	// payment date
	PaymentDate nulls.Time `json:"payment_date,omitempty"`

	// total payout
	TotalPayout int `json:"total_payout,omitempty"`
}

// swagger:model
type ClaimCreateInput struct {
	// event date
	EventDate time.Time `json:"event_date"`

	// event type, one of: Theft, Impact, Lightning, Water, Evacuation, Other
	EventType ClaimEventType `json:"event_type"`

	// event description
	EventDescription string `json:"event_description"`
}
