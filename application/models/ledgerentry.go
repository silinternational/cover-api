package models

import (
	"fmt"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/fin"
)

type LedgerEntryType string

func (t LedgerEntryType) IsClaim() bool {
	if t == LedgerEntryTypeClaim || t == LedgerEntryTypeClaimAdjustment {
		return true
	}
	return false
}

const (
	LedgerEntryTypeNewCoverage      = LedgerEntryType("NewCoverage")
	LedgerEntryTypeCoverageChange   = LedgerEntryType("CoverageChange")
	LedgerEntryTypePolicyAdjustment = LedgerEntryType("PolicyAdjustment")
	LedgerEntryTypeClaim            = LedgerEntryType("Claim")
	LedgerEntryTypeLegacy5          = LedgerEntryType("5")
	LedgerEntryTypeClaimAdjustment  = LedgerEntryType("ClaimAdjustment")
	LedgerEntryTypeLegacy20         = LedgerEntryType("20")
)

var ValidLedgerEntryTypes = map[LedgerEntryType]struct{}{
	LedgerEntryTypeNewCoverage:      {},
	LedgerEntryTypeCoverageChange:   {},
	LedgerEntryTypePolicyAdjustment: {},
	LedgerEntryTypeClaim:            {},
	LedgerEntryTypeLegacy5:          {},
	LedgerEntryTypeClaimAdjustment:  {},
	LedgerEntryTypeLegacy20:         {},
}

type LedgerEntries []LedgerEntry

type LedgerEntry struct {
	ID uuid.UUID `db:"id"`

	PolicyID         uuid.UUID       `db:"policy_id"`
	ItemID           nulls.UUID      `db:"item_id"`
	ClaimID          nulls.UUID      `db:"claim_id"`
	EntityCode       string          `db:"entity_code"`
	RiskCategoryName string          `db:"risk_category_name"`
	RiskCategoryCC   string          `db:"risk_category_cc"`
	Type             LedgerEntryType `db:"type" validate:"ledgerEntryType"`
	PolicyType       api.PolicyType  `db:"policy_type" validate:"policyType"`
	HouseholdID      string          `db:"household_id"`
	CostCenter       string          `db:"cost_center"`
	AccountNumber    string          `db:"account_number"`
	FirstName        string          `db:"first_name"`
	LastName         string          `db:"last_name"`
	Amount           int             `db:"amount"`
	DateSubmitted    time.Time       `db:"date_submitted"`
	DateEntered      nulls.Time      `db:"date_entered"`
	LegacyID         nulls.Int       `db:"legacy_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (le *LedgerEntry) Create(tx *pop.Connection) error {
	return create(tx, le)
}

// AllForMonth returns all the non-entered entries (date_entered is null) for the month.
// The provided date must be the first day of the month.
func (le *LedgerEntries) AllForMonth(tx *pop.Connection, firstDay time.Time) error {
	lastDay := domain.EndOfMonth(firstDay)

	err := tx.Where("date_submitted BETWEEN ? and ?", firstDay, lastDay).
		Where("date_entered IS NULL").All(le)

	return appErrorFromDB(err, api.ErrorQueryFailure)
}

type TransactionBlocks map[string]LedgerEntries // keyed by account

func (le *LedgerEntries) ToCsv(batchDate time.Time) []byte {
	sage := fin.NewBatch(fin.ProviderTypeSage, batchDate)

	blocks := le.MakeBlocks()
	for account, ledgerEntries := range blocks {
		if len(ledgerEntries) == 0 {
			continue
		}
		var balance int
		for _, l := range ledgerEntries {
			sage.AppendToBatch(fin.Transaction{
				Account:     domain.Env.ExpenseAccount,
				Amount:      l.Amount,
				Description: l.transactionDescription(),
				Reference:   l.transactionReference(),
				Date:        l.DateSubmitted,
			})

			balance -= l.Amount
		}
		sage.AppendToBatch(fin.Transaction{
			Account:     account,
			Amount:      balance,
			Description: ledgerEntries[0].balanceDescription(),
			Reference:   "",
			Date:        batchDate,
		})
	}

	return sage.BatchToCSV()
}

func (le *LedgerEntries) MakeBlocks() TransactionBlocks {
	blocks := TransactionBlocks{}
	for _, e := range *le {
		account := e.getIncomeAccount()
		blocks[account] = append(blocks[account], e)
	}
	return blocks
}

// getIncomeAccount maps the ledger data to an income account for billing
func (le *LedgerEntry) getIncomeAccount() string {
	// TODO: move hard-coded account numbers to the database or to environment variables
	account := ""

	if le.Type.IsClaim() {
		account = "63550"
	} else {
		switch le.EntityCode {
		case "", "MMB/STM":
			account = "40200"
		case "SIL":
			account = "43250"
		default:
			account = "44250"
		}
	}

	incomeAccount := account + le.RiskCategoryCC

	return incomeAccount
}

// TODO: make a better description format unless it has to be the same as before (which I doubt)
func (le *LedgerEntry) transactionDescription() string {
	dateString := le.DateSubmitted.Format("Jan 02, 2006")

	description := ""
	// TODO: change this to api.PolicyTypeHousehold (requires a database update)
	if le.EntityCode == "MMB/STM" {
		description = fmt.Sprintf("%s,%s %s %s %s",
			le.LastName, le.FirstName, le.RiskCategoryName, le.Type, dateString)
	} else {
		description = fmt.Sprintf("%s %s (%s) %s",
			le.RiskCategoryName, le.Type, le.CostCenter, dateString)
	}

	return fmt.Sprintf("%.60s", description) // truncate to Sage limit of 60 characters
}

// Merge HouseholdID and CostCenter into one column
func (le *LedgerEntry) transactionReference() string {
	// TODO: change this to api.PolicyTypeHousehold (requires a database update)
	if le.EntityCode == "MMB/STM" {
		return fmt.Sprintf("MC %s", le.HouseholdID)
	}
	return le.CostCenter
}

func (le *LedgerEntry) balanceDescription() string {
	premiumsOrClaims := "Premiums"
	if le.Type == LedgerEntryTypeClaim || le.Type == LedgerEntryTypeClaimAdjustment {
		premiumsOrClaims = "Claims"
	}
	return fmt.Sprintf("Total %s %s %s", le.EntityCode, le.RiskCategoryName, premiumsOrClaims)
}

// NewLedgerEntry creates a basic LedgerEntry with common fields completed.
// Requires pre-hydration of policy.EntityCode. If item is not nil, item.RiskCategory must be hydrated.
func NewLedgerEntry(policy Policy, item *Item, claim *Claim) LedgerEntry {
	costCenter := ""
	if policy.Type == api.PolicyTypeCorporate {
		costCenter = policy.CostCenter + " / " + policy.AccountDetail
	}
	le := LedgerEntry{
		PolicyID:      policy.ID,
		PolicyType:    policy.Type,
		EntityCode:    policy.EntityCode.Code,
		DateSubmitted: time.Now().UTC(),
		AccountNumber: policy.Account,
		CostCenter:    costCenter,
		HouseholdID:   policy.HouseholdID.String,
	}
	if item != nil {
		le.ItemID = nulls.NewUUID(item.ID)
		le.RiskCategoryName = item.RiskCategory.Name
		le.RiskCategoryCC = item.RiskCategory.CostCenter
	}
	if claim != nil {
		le.ClaimID = nulls.NewUUID(claim.ID)
	}
	return le
}
