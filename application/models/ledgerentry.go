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

type LedgerEntries []LedgerEntry

type LedgerEntry struct {
	ID uuid.UUID `db:"id"`

	PolicyID      uuid.UUID  `db:"policy_id"`
	ItemID        nulls.UUID `db:"item_id"`
	EntityCodeID  nulls.UUID `db:"entity_code_id"`
	Amount        int        `db:"amount"`
	DateSubmitted time.Time  `db:"date_submitted"`
	DateEntered   nulls.Time `db:"date_entered"`

	// The following fields are primarily for legacy data and may not be needed long-term
	// However, some may be useful as a permanent record in case policies change...TBD.
	LegacyID           nulls.Int `db:"legacy_id"`
	RecordType         nulls.Int `db:"record_type"`
	PolicyType         nulls.Int `db:"policy_type"`
	AccountNumber      string    `db:"account_number"`
	AccountCostCenter1 string    `db:"account_cost_center1"`
	AccountCostCenter2 string    `db:"account_cost_center2"`
	EntityCode         string    `db:"entity_code"`
	FirstName          string    `db:"first_name"`
	LastName           string    `db:"last_name"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (le *LedgerEntry) Create(tx *pop.Connection) error {
	return create(tx, le)
}

func (le *LedgerEntries) FindBatch(tx *pop.Connection, firstDay time.Time) error {
	lastDay := domain.EndOfMonth(firstDay)

	err := tx.Where("date_submitted BETWEEN ? and ?", firstDay, lastDay).
		Where("date_entered IS NULL").
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
				Amount:      -l.Amount,
				Description: l.transactionDescription(),
				Reference:   l.transactionReference(),
				Date:        batchDate, // can be any date in the batch month
			})

			balance += l.Amount
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
		if e.RecordType.Valid && e.RecordType.Int == 4 {
			account = "63550" // Claims
		} else {
			account = accountMap[e.EntityCode]
		}

		if e.PolicyType.Valid && e.PolicyType.Int == 2 {
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
	ccrOrPP := le.policyTypeName()

	recordType := "????"
	// TODO: change RecordType column in the database to string with restricted options
	switch le.RecordType.Int {
	case 1:
		recordType = "New coverage"
	case 2:
		recordType = "Coverage change"
	case 4:
		recordType = "Claim"
	}

	dateString := le.DateSubmitted.Format("Jan 02, 2006")

	description := ""
	// TODO: change this to api.PolicyTypeHousehold (requires a database update)
	if le.EntityCode == "MMB/STM" {
		// TODO: figure out why names are blank in legacy data AND ensure they are not blank in new data
		description = fmt.Sprintf("%s,%s %s %s %s", le.LastName, le.FirstName, ccrOrPP, recordType, dateString)
	} else {
		description = fmt.Sprintf("%s %s (%s) %s", ccrOrPP, recordType, le.AccountCostCenter2, dateString)
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
	if le.RecordType.Int == 4 {
		premiumsOrClaims = "Claims"
	}
	return fmt.Sprintf("Total %s %s %s", le.EntityCode, le.policyTypeName(), premiumsOrClaims)
}

func (le *LedgerEntry) policyTypeName() string {
	ccrOrPP := "CCR"
	if le.PolicyType.Valid && le.PolicyType.Int == 2 {
		ccrOrPP = "PP"
	}
	return ccrOrPP
}

func getFiscalPeriod(month int) int {
	return (month-domain.Env.FiscalStartMonth+12)%12 + 1
}
