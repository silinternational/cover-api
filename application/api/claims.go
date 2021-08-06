package api

import (
	"time"

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

type Claims []Claim

type Claim struct {
	ID               uuid.UUID      `json:"id"`
	PolicyID         uuid.UUID      `json:"policy_id"`
	EventDate        time.Time      `json:"event_date"`
	EventType        ClaimEventType `json:"event_type"`
	EventDescription string         `json:"event_description"`
	Status           ClaimStatus    `json:"status"`
	ReviewDate       time.Time      `json:"review_date,omitempty"`
	ReviewerID       uuid.UUID      `json:"reviewer_id,omitempty"`
	PaymentDate      time.Time      `json:"payment_date,omitempty"`
	TotalPayout      int            `json:"total_payout,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}
