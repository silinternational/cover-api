package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

// PolicyType
//
// may be one of: Household, Team
//
// swagger:model
type PolicyType string

const (
	PolicyTypeHousehold = PolicyType("Household")
	PolicyTypeTeam      = PolicyType("Team")
)

// swagger:model
type Policies []Policy

// Policy represents a single policy, either household or team
// swagger:model
type Policy struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// policy name
	Name string `json:"name"`

	// policy type
	Type PolicyType `json:"type"`

	// Household ID for billing
	HouseholdID string `json:"household_id,omitempty"`

	// Cost center for billing
	CostCenter string `json:"cost_center,omitempty"`

	// Account code for billing
	Account string `json:"account,omitempty"`

	// AccountDetail allows for optional detail to route transactions. e.g.: "Nigeria Grp Off-Ins"
	AccountDetail string `json:"account_detail,omitempty"`

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

// PolicyCreate represents payload for creating a policy
// swagger:model
type PolicyCreate struct {
	// policy name
	Name string `json:"name"`

	// Policy type. Only needed for steward endpoints. For customers, this will be set by the api.
	Type string `json:"type"`

	// Household ID for billing. Only required/allowed on Household type policies.
	HouseholdID nulls.String `json:"household_id,omitempty"`

	// Cost center for billing. Only required/allowed on Team type policies.
	CostCenter string `json:"cost_center,omitempty"`

	// Account code for billing. Only required/allowed on Team type policies.
	Account string `json:"account,omitempty"`

	// AccountDetail allows for optional detail to route transactions. e.g.: "Nigeria Grp Off-Ins"
	AccountDetail string `json:"account_detail,omitempty"`

	// Entity code for billing. Only required/allowed on Team type policies.
	EntityCode string `json:"entity_code,omitempty"`
}

// PolicyUpdate represents payload for updating a policy
// swagger:model
type PolicyUpdate struct {
	// policy name
	Name string `json:"name"`

	// Household ID for billing. Only required/allowed on Household type policies.
	HouseholdID nulls.String `json:"household_id,omitempty"`

	// Cost center for billing. Only required/allowed on Team type policies.
	CostCenter string `json:"cost_center,omitempty"`

	// Account code for billing. Only required/allowed on Team type policies.
	Account string `json:"account,omitempty"`

	// AccountDetail allows for optional detail to route transactions. e.g.: "Nigeria Grp Off-Ins"
	AccountDetail string `json:"account_detail,omitempty"`

	// Entity code for billing. Only required/allowed on Team type policies.
	EntityCode string `json:"entity_code,omitempty"`
}
