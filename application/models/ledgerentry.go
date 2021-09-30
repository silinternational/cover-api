package models

import (
	"errors"
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
	Amount           int                   `db:"amount"`
	DateSubmitted    time.Time             `db:"date_submitted"`
	DateEntered      nulls.Time            `db:"date_entered"`

	// The following fields are primarily for legacy data and may not be needed long-term
	// However, some may be useful as a permanent record in case policies change...TBD.
	LegacyID           int    `db:"legacy_id"`
	AccountNumber      string `db:"account_number"`
	AccountCostCenter1 string `db:"account_cost_center1"`
	AccountCostCenter2 string `db:"account_cost_center2"`
	FirstName          string `db:"first_name"`
	LastName           string `db:"last_name"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (le *LedgerEntry) Create(tx *pop.Connection) error {
	return create(tx, le)
}

func (le *LedgerEntries) FindBatch(tx *pop.Connection, firstDay time.Time) error {
	lastDay := domain.EndOfMonth(firstDay)

	err := tx.Where("date_submitted BETWEEN ? and ?", firstDay, lastDay).
		// TODO: Temporarily hardcoded to a month with closed transactions. Add this WHERE clause before going "live".
		// Where("date_entered IS NULL").
		All(le)

	return appErrorFromDB(err, api.ErrorQueryFailure)
}

type TransactionBlocks map[string]LedgerEntries // keyed by account

func (le *LedgerEntries) ToCsv(batchDate time.Time) ([]byte, error) {
	if len(*le) == 0 {
		return nil, errors.New("no ledger entries, cannot convert to CSV")
	}

	date := (*le)[0].DateSubmitted
	sage := fin.Sage{
		Year:               date.Year(),
		Period:             getFiscalPeriod(int(date.Month())),
		JournalDescription: date.Format("January 2006 MAP JE"),
	}

	blocks := le.MakeBlocks()
	for account, ledgerEntries := range blocks {
		if len(ledgerEntries) == 0 {
			continue
		}
		var balance int
		for _, l := range ledgerEntries {
			sage.AppendToBatch(fin.Transaction{
				Account:     "19349MMAP12",
				Amount:      l.Amount,
				Description: l.transactionDescription(),
				Reference:   l.transactionReference(),
				Date:        batchDate, // can be any date in the batch month
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
	accountMap := map[string]string{
		"MMB/STM": "40200",
		"SIL":     "43250",
		"WBT":     "44250",
	}
	for _, e := range *le {
		var account string
		if e.RecordType == LedgerEntryRecordTypeClaim || e.RecordType == LedgerEntryRecordTypeClaimAdjustment {
			account = "63550" // Claims
		} else {
			account = accountMap[e.EntityCode]
		}

		if e.RiskCategoryName == "Stationary" {
			account += "MPRO12"
		} else {
			account += "MCMC12"
		}

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
		// TODO: figure out why names are blank in legacy data AND ensure they are not blank in new data
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

func getFiscalPeriod(month int) int {
	return (month-domain.Env.FiscalStartMonth+12)%12 + 1
}
