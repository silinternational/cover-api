package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

// ClaimIncidentType
//
// may be one of: Theft, Impact, Electrical Surge, Water Damage, Evacuation, Other
//
// swagger:model
type ClaimIncidentType string

// IsRepairable answers the question "Are items with this incident type potentially repairable?"
func (c ClaimIncidentType) IsRepairable() bool {
	for _, cit := range AllClaimIncidentTypes {
		if cit.Name == c {
			return cit.IsRepairable
		}
	}
	return false
}

// ClaimStatus
//
// may be one of: Draft, Review1, Review2, Review3, Revision, Receipt, Approved, Paid, Denied
//
// swagger:model
type ClaimStatus string

func (s ClaimStatus) WasReviewed() bool {
	switch s {
	case ClaimStatusDenied, ClaimStatusRevision, ClaimStatusReceipt,
		ClaimStatusApproved, ClaimStatusPaid, ClaimStatusReview3:
		return true
	}
	return false
}

const (
	ClaimIncidentTypeTheft           = ClaimIncidentType("Theft")
	ClaimIncidentTypeImpact          = ClaimIncidentType("Impact")
	ClaimIncidentTypeElectricalSurge = ClaimIncidentType("Electrical Surge")
	ClaimIncidentTypeWaterDamage     = ClaimIncidentType("Water Damage")
	ClaimIncidentTypeEvacuation      = ClaimIncidentType("Evacuation")
	ClaimIncidentTypeOther           = ClaimIncidentType("Other")
)

// swagger:model
type ClaimIncidentTypeStruct struct {
	Name         ClaimIncidentType `json:"name"`
	IsRepairable bool              `json:"is_repairable"`
}

var AllClaimIncidentTypes = []ClaimIncidentTypeStruct{
	{ClaimIncidentTypeTheft, false},
	{ClaimIncidentTypeImpact, true},
	{ClaimIncidentTypeElectricalSurge, true},
	{ClaimIncidentTypeWaterDamage, true},
	{ClaimIncidentTypeEvacuation, false},
	{ClaimIncidentTypeOther, true},
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
	// human friendly seven character string
	// example: AB43312
	ReferenceNumber string `json:"reference_number"`

	// incident date
	//
	// swagger:strfmt date-time
	IncidentDate time.Time `json:"incident_date"`

	// incident type
	IncidentType ClaimIncidentType `json:"incident_type"`

	// incident description .
	IncidentDescription string `json:"incident_description"`

	// incident status
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

	// total payout (0.01 USD)
	TotalPayout Currency `json:"total_payout,omitempty"`

	// message from a reviewer detailing the revisions needed
	StatusReason string `json:"status_reason"`

	MayPerform map[string]bool `json:"may_perform"`

	// list of items included in claim
	Items ClaimItems `json:"claim_items"`

	// list of files attached to the claim
	Files []ClaimFile `json:"claim_files"`
}

// swagger:model
type RecentClaims []RecentClaim

// swagger:model
type RecentClaim struct {
	// The time the claim had its status changed
	// swagger:strfmt date-time
	StatusUpdatedAt time.Time

	Claim Claim
}

// swagger:model
type ClaimCreateInput struct {
	// incident date
	IncidentDate time.Time `json:"incident_date"`

	IncidentType ClaimIncidentType `json:"incident_type"`

	// incident description
	IncidentDescription string `json:"incident_description"`
}

// swagger:model
type ClaimUpdateInput struct {
	// incident date
	IncidentDate time.Time `json:"incident_date"`

	IncidentType ClaimIncidentType `json:"incident_type"`

	// incident description
	IncidentDescription string `json:"incident_description"`
}

// swagger:model
type ClaimStatusInput struct {
	// message from a reviewer noting the reason for the new status, e.g. detailing the revisions needed
	StatusReason string `json:"status_reason"`
}
