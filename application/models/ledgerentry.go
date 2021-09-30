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

type LedgerEntryRecordType string

const (
	LedgerEntryRecordTypeNewCoverage      = LedgerEntryRecordType("NewCoverage")
	LedgerEntryRecordTypeCoverageChange   = LedgerEntryRecordType("CoverageChange")
	LedgerEntryRecordTypePolicyAdjustment = LedgerEntryRecordType("PolicyAdjustment")
	LedgerEntryRecordTypeClaim            = LedgerEntryRecordType("Claim")
	LedgerEntryRecordTypeLegacy5          = LedgerEntryRecordType("5")
	LedgerEntryRecordTypeClaimAdjustment  = LedgerEntryRecordType("ClaimAdjustment")
	LedgerEntryRecordTypeLegacy20         = LedgerEntryRecordType("20")
)

var ValidLedgerEntryRecordTypes = map[LedgerEntryRecordType]struct{}{
	LedgerEntryRecordTypeNewCoverage:      {},
	LedgerEntryRecordTypeCoverageChange:   {},
	LedgerEntryRecordTypePolicyAdjustment: {},
	LedgerEntryRecordTypeClaim:            {},
	LedgerEntryRecordTypeLegacy5:          {},
	LedgerEntryRecordTypeClaimAdjustment:  {},
	LedgerEntryRecordTypeLegacy20:         {},
}

type LedgerEntries []LedgerEntry

type LedgerEntry struct {
	ID uuid.UUID `db:"id"`

	PolicyID         uuid.UUID             `db:"policy_id"`
	ItemID           nulls.UUID            `db:"item_id"`
	EntityCode       string                `db:"entity_code"`
	RiskCategoryName string                `db:"risk_category_name"`
	RecordType       LedgerEntryRecordType `db:"record_type" validate:"ledgerEntryRecordType"`
	IncomeAccount    string                `db:"income_account"`
	FirstName        string                `db:"first_name"`
	LastName         string                `db:"last_name"`
	Amount           int                   `db:"amount"`
	DateSubmitted    time.Time             `db:"date_submitted"`
	DateEntered      nulls.Time            `db:"date_entered"`

	// The following fields are primarily for legacy data and may not be needed long-term
	// However, some may be useful as a permanent record in case policies change...TBD.
	LegacyID           nulls.Int `db:"legacy_id"`
	AccountNumber      string    `db:"account_number"`
	AccountCostCenter1 string    `db:"account_cost_center1"` // TODO: rename to HouseholdID
	AccountCostCenter2 string    `db:"account_cost_center2"` // TODO: rename to CostCenter

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (le *LedgerEntry) Create(tx *pop.Connection) error {
	return create(tx, le)
}

func (le *LedgerEntries) AllForMonth(tx *pop.Connection, firstDay time.Time) error {
	lastDay := domain.EndOfMonth(firstDay)

	err := tx.Where("date_submitted BETWEEN ? and ?", firstDay, lastDay).
		Where("date_entered IS NULL").All(le)

	return appErrorFromDB(err, api.ErrorQueryFailure)
}

type TransactionBlocks map[string]LedgerEntries // keyed by account

func (le *LedgerEntries) ToCsv(batchDate time.Time) []byte {
	sage := fin.NewBatch("sage", batchDate)

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
		account := e.IncomeAccount
		blocks[account] = append(blocks[account], e)
	}
	return blocks
}

// TODO: make a better description format unless it has to be the same as before (which I doubt)
func (le *LedgerEntry) transactionDescription() string {
	dateString := le.DateSubmitted.Format("Jan 02, 2006")

	description := ""
	// TODO: change this to api.PolicyTypeHousehold (requires a database update)
	if le.EntityCode == "MMB/STM" {
		description = fmt.Sprintf("%s,%s %s %s %s",
			le.LastName, le.FirstName, le.RiskCategoryName, le.RecordType, dateString)
	} else {
		description = fmt.Sprintf("%s %s (%s) %s",
			le.RiskCategoryName, le.RecordType, le.AccountCostCenter2, dateString)
	}

	return fmt.Sprintf("%.60s", description) // truncate to Sage limit of 60 characters
}

// Merge AccountCostCenter1 and AccountCostCenter2 into one column
func (le *LedgerEntry) transactionReference() string {
	// TODO: change this to api.PolicyTypeHousehold (requires a database update)
	if le.EntityCode == "MMB/STM" {
		return fmt.Sprintf("MC %s", le.AccountCostCenter1)
	}
	return le.AccountCostCenter2
}

func (le *LedgerEntry) balanceDescription() string {
	premiumsOrClaims := "Premiums"
	if le.RecordType == LedgerEntryRecordTypeClaim || le.RecordType == LedgerEntryRecordTypeClaimAdjustment {
		premiumsOrClaims = "Claims"
	}
	return fmt.Sprintf("Total %s %s %s", le.EntityCode, le.RiskCategoryName, premiumsOrClaims)
}
