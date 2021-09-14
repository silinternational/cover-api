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

// ClaimStatus
//
// may be one of: Draft, Pending, Approved, Denied
//
// swagger:model
type ClaimStatus string

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
	// human friendly seven character string
	// example: AB43312
	ReferenceNumber string `json:"reference_number"`

	// Incident date
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

	// total payout
	TotalPayout int `json:"total_payout,omitempty"`

	// list of items included in claim
	Items ClaimItems `json:"claim_items"`

	// list of files attached to the claim
	Files []ClaimFile `json:"claim_files"`
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
