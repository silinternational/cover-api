package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

// PolicyType
//
// may be one of: Household, Corporate
//
// swagger:model
type PolicyType string

const (
	PolicyTypeHousehold = PolicyType("Household")
	PolicyTypeCorporate = PolicyType("Corporate")
)

// swagger:model
type Policies []Policy

// Policy represents a single policy, either household or corporate
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
	EntityCode EntityCode `json:"entity_code,omitempty"`

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

// PolicyCreate represents payload for updating a policy
// swagger:model
type PolicyCreate struct {
	// Policy type. Only needed for steward endpoints. For customers, this will be set by the api.
	Type string `json:"type"`

	// Household ID for billing. Only required/allowed on Household type policies.
	HouseholdID nulls.String `json:"household_id,omitempty"`

	// Cost center for billing. Only required/allowed on Corporate type policies.
	CostCenter string `json:"cost_center,omitempty"`

	// Account code for billing. Only required/allowed on Corporate type policies.
	Account string `json:"account,omitempty"`

	// Entity code for billing. Only required/allowed on Corporate type policies.
	EntityCode string `json:"entity_code,omitempty"`
}

// PolicyUpdate represents payload for updating a policy
// swagger:model
type PolicyUpdate struct {
	// Household ID for billing. Only required/allowed on Household type policies.
	HouseholdID nulls.String `json:"household_id,omitempty"`

	// Cost center for billing. Only required/allowed on Corporate type policies.
	CostCenter string `json:"cost_center,omitempty"`

	// Account code for billing. Only required/allowed on Corporate type policies.
	Account string `json:"account,omitempty"`

	// Entity code for billing. Only required/allowed on Corporate type policies.
	EntityCode string `json:"entity_code,omitempty"`
}
