package api

import (
	"time"

	"github.com/gofrs/uuid"
)

type PolicyType string

const (
	PolicyTypeHousehold = PolicyType("Household")
	PolicyTypeOU        = PolicyType("OU")
)

type Policies []Policy

// Policy represents a single policy, either household or ou
// swagger:model
type Policy struct {
	// unique id (uuid) for thread
	//
	// swagger:strfmt uuid4
	// unique: true
	// example: 63d5b060-1460-4348-bdf0-ad03c105a8d5
	ID uuid.UUID `json:"id"`

	// policy type
	// required: true
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
	CreatedAt time.Time `json:"created_at"`

	// The time the policy was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// List of policy members
	Members PolicyMembers `json:"members"`

	// List of dependents on policy
	Dependents PolicyDependents `json:"dependents,omitempty"`
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
