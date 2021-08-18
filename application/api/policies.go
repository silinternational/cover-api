package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// PolicyType
//
// may be one of: Household, OU
//
// swagger:model
type PolicyType string

const (
	PolicyTypeHousehold = PolicyType("Household")
	PolicyTypeOU        = PolicyType("OU")
)

// swagger:model
type Policies []Policy

// Policy represents a single policy, either household or OU
// swagger:model
type Policy struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// policy type
	Type PolicyType `json:"type"`

	// Household ID for billing
	HouseholdID string `json:"household_id,omitempty"`

	// Cost center for billing
	CostCenter string `json:"cost_center,omitempty"`

	// Account code for billing
	Account string `json:"account,omitempty"`

	// Entity code for billing
	EntityCode string `json:"entity_code,omitempty"`

	// The time the policy was created
	//
	// swagger:strfmt date-time
	CreatedAt time.Time `json:"created_at"`

	// The time the policy was last updated
	//
	// swagger:strfmt date-time
	UpdatedAt time.Time `json:"updated_at"`

	// List of policy members
	Members PolicyMembers `json:"members"`

	// List of dependents on policy
	Dependents PolicyDependents `json:"dependents,omitempty"`

	// List of claims on policy
	Claims Claims `json:"claims,omitempty"`
}

// PolicyUpdate represents payload for updating a policy
// swagger:model
type PolicyUpdate struct {
	// Household ID for billing
	HouseholdID string `json:"household_id,omitempty"`

	// Cost center for billing
	CostCenter string `json:"cost_center,omitempty"`

	// Account code for billing
	Account string `json:"account,omitempty"`

	// Entity code for billing
	EntityCode string `json:"entity_code,omitempty"`
}
