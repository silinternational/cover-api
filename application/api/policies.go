package api

import (
	"time"

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
	PolicyStatusActive  = "Active"
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
	HouseholdID string `json:"household_id"`

	// Cost center for billing
	CostCenter string `json:"cost_center"`

	// Account code for billing
	Account string `json:"account"`

	// AccountDetail allows for optional detail to route transactions. e.g.: "Nigeria Grp Off-Ins"
	AccountDetail string `json:"account_detail"`

	// Entity code for billing
	EntityCode EntityCode `json:"entity_code"`

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
	Dependents PolicyDependents `json:"dependents"`

	// List of invites for this policy
	Invites PolicyUserInvites `json:"invites"`

	// List of claims on policy
	Claims Claims `json:"claims"`

	// List of strikes on policy
	Strikes Strikes `json:"strikes"`

	// List of LedgerReports -- only available on PoliciesView
	LedgerReports LedgerReports `json:"ledger_reports"`
}

// PolicyCreate represents payload for creating a policy
// swagger:model
type PolicyCreate struct {
	// policy name
	Name string `json:"name"`

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
	HouseholdID *string `json:"household_id,omitempty"`

	// Cost center for billing. Only required/allowed on Team type policies.
	CostCenter string `json:"cost_center,omitempty"`

	// Account code for billing. Only required/allowed on Team type policies.
	Account string `json:"account,omitempty"`

	// AccountDetail allows for optional detail to route transactions. e.g.: "Nigeria Grp Off-Ins"
	AccountDetail string `json:"account_detail,omitempty"`

	// Entity code for billing. Only required/allowed on Team type policies.
	EntityCode string `json:"entity_code,omitempty"`
}

// swagger:model
type PolicyLedgerReportCreateInput struct {
	// Report types:
	// + `Monthly` - Return all ledger entries not yet reconciled, up to the beginning of the given date.
	// + `Annual` - Return the policy renewal entries for the year of the given date.
	//
	Type string `json:"type"`

	// Report month, e.g. return the policy's ledger entries entered in that month and year.
	//  The month and year (together) must not be in the future.
	//  For annual reports only, the month may be 0.
	Month int `json:"month"`

	// Report year, e.g. return the policy's ledger entries entered in that year.
	Year int `json:"year"`
}

// swagger:model
type PoliciesImportResponse struct {
	LinesProcessed  int `json:"lines_processed"`
	PoliciesCreated int `json:"policies_created"`
	ItemsCreated    int `json:"items_created"`
}
