package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

type LedgerEntryType string

// swagger:model
type BatchApproveResponse struct {
	NumberOfRecordsApproved int `json:"number_of_records_approved"`
}

// swagger:model
type LedgerReconcileInput struct {
	EndDate string `json:"end_date"`
}

// swagger:model
type LedgerReports []LedgerReport

// swagger:model
type LedgerReport struct {
	ID        uuid.UUID `json:"id"`
	File      File      `json:"file"`
	Type      string    `json:"type"`
	Date      time.Time `json:"date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// swagger:model
type LedgerReportCreateInput struct {
	// Report types:
	// + `monthly` - Return all ledger entries not yet reconciled, up to the beginning of the given date.
	// + `annual` - Return the policy renewal entries for the year of the given date.
	//
	Type string `json:"type"`

	// Report date, e.g. return the ledger entries prior to the given date. Details vary by the report type.
	Date string `json:"date"`
}

// swagger:model
type LedgerEntries []LedgerEntry

// swagger:model
type LedgerEntry struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// policy ID
	//
	// swagger:strfmt uuid4
	PolicyID uuid.UUID `json:"policy_id"`

	// item ID
	//
	// swagger:strfmt uuid4
	ItemID nulls.UUID `json:"item_id"`

	// claim ID
	//
	// swagger:strfmt uuid4
	ClaimID          nulls.UUID      `json:"claim_id"`
	EntityCode       string          `json:"entity_code"`
	RiskCategoryName string          `json:"risk_category_name"`
	RiskCategoryCC   string          `json:"risk_category_cc"` // Risk Category Cost Center
	Type             LedgerEntryType `json:"type"`
	PolicyType       PolicyType      `json:"policy_type"`
	HouseholdID      string          `json:"household_id"`
	CostCenter       string          `json:"cost_center"`
	AccountNumber    string          `json:"account_number"`
	IncomeAccount    string          `json:"income_account"`

	// name of accountable person if available, otherwise the policy name
	Name   string   `json:"name"`
	Amount Currency `json:"amount"`

	// date added to ledger
	//
	// swagger:strfmt date-time
	DateSubmitted time.Time `json:"date_submitted"`

	// date entered into accounting system
	//
	// swagger:strfmt date-time
	DateEntered *time.Time `json:"date_entered"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
