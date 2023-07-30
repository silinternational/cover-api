package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

type LedgerEntryType string

// swagger:model
type LedgerReports []LedgerReport

// swagger:model
type LedgerReport struct {
	ID               uuid.UUID `json:"id"`
	File             File      `json:"file"`
	Type             string    `json:"type"`
	Date             time.Time `json:"date"`
	TransactionCount int       `json:"transaction_count"`
	IsCleared        bool      `json:"is_cleared"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
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
type LedgerTable struct {
	LastChanged     time.Time `json:"last_changed"`
	CoverageValue   Currency  `json:"coverage_value"`
	PremiumRate     float64   `json:"premium_rate"`
	PremiumTotal    Currency  `json:"premium_total"`
	PayoutTotal     Currency  `json:"payout_total"`
	NetTransactions Currency  `json:"net_transactions"`
	ReportMonth     int       `json:"report_month"`
	ReportYear      int       `json:"report_year"`

	Entries []LedgerTableEntry `json:"entries"`
}

// swagger:model
type LedgerTableEntry struct {
	ItemName     string    `json:"item_name"`
	StatusBefore string    `json:"status_before"`
	StatusAfter  string    `json:"status_after"`
	Type         string    `json:"type"`
	Value        Currency  `json:"value"`
	Date         time.Time `json:"date"`
	AssignedTo   string    `json:"assigned_to"`
	Location     string    `json:"location"`
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

	// name of accountable person if available
	Name string `json:"name"`

	PolicyName string `json:"policy_name"`

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

// swagger:model
type AnnualRenewalStatus struct {
	IsComplete     bool `json:"is_complete"`
	ItemsToProcess int  `json:"items_to_process"`
}
