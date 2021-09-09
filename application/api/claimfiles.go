package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// ClaimFilePurpose
//
// may be one of: "Receipt", "Evidence of FMV", "Repair Estimate"
//
// swagger:model
type ClaimFilePurpose string

const (
	ClaimFilePurposeReceipt        = ClaimFilePurpose("Receipt")
	ClaimFilePurposeEvidenceOfFMV  = ClaimFilePurpose("Evidence of FMV")
	ClaimFilePurposeRepairEstimate = ClaimFilePurpose("Repair Estimate")
)

// swagger:model
type ClaimFile struct {
	// ID of the ClaimFile
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// ID of the Claim
	//
	// swagger:strfmt uuid4
	ClaimID uuid.UUID `json:"claim_id"`

	// ID of the File
	//
	// swagger:strfmt uuid4
	FileID uuid.UUID `json:"file_id"`

	// Purpose of file
	Purpose ClaimFilePurpose `json:"purpose"`

	// created time
	//
	// swagger:strfmt date-time
	CreatedAt time.Time `json:"created_at"`

	// last updated time
	//
	// swagger:strfmt date-time
	UpdatedAt time.Time `json:"updated_at"`

	// file object
	File File `json:"file"`
}

// swagger:model
type ClaimFileAttachInput struct {
	// File ID to attach to the claim
	//
	// swagger:strfmt uuid4
	FileID uuid.UUID `json:"file_id"`

	// Purpose of file
	Purpose ClaimFilePurpose `json:"purpose"`
}
