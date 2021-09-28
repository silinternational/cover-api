package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// RiskCategories is a slice of RiskCategory objects
// swagger:model
type RiskCategories []RiskCategory

// RiskCategory represents an item category's risk category
// swagger:model
type RiskCategory struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// risk category name
	Name string `json:"name"`

	// maximum coverage per policy (0.01 USD)
	PolicyMax int `json:"policy_max"`

	// financial cost center code
	CostCenter string `json:"cost_center"`

	// created date
	//
	// swagger:strfmt date-time
	CreatedAt time.Time `json:"created_at"`

	// updated date
	//
	// swagger:strfmt date-time
	UpdatedAt time.Time `json:"updated_at"`
}
