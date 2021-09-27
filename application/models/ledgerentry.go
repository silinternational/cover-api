package models

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
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
		// TODO: re-enable this part of the query to prevent duplicate entries
		// Where("date_entered IS NULL").
		All(le)

	return appErrorFromDB(err, api.ErrorQueryFailure)
}

func (le *LedgerEntries) ToCsv(batchDate time.Time) ([]byte, error) {
	if len(*le) == 0 {
		return nil, errors.New("no ledger entries, cannot convert to CSV")
	}

	const header1 = `"RECTYPE","BATCHID","BTCHENTRY","ORIGCOMP","SRCELEDGER","SRCETYPE","FSCSYR","FSCSPERD","SWEDIT","JRNLDESC","REVPERD","ERRBATCH","ERRENTRY","DETAILCNT","PROCESSCMD"` + "\n"
	const header2 = `"RECTYPE","BATCHNBR","JOURNALID","TRANSNBR","DESCOMP","ROUTE","ACCTID","COMPANYID","TRANSAMT","SCURNDEC","TRANSDESC","TRANSREF","TRANSDATE","SRCELDGR","SRCETYPE",` +
		`"record_type","entity_code","policy_type","cost_center_1","cost_center_2"` + // TODO: remove these columns
		"\n"

	rows := [][]byte{[]byte(header2)}

	// TODO: put this back
	// rows := [][]byte{[]byte(header1), []byte(header2)}
	// rows = append(rows, (*le)[0].csvSummaryRow())

	for i, l := range *le {
		newRow := l.sageTransactionRow(i, batchDate)
		rows = append(rows, newRow)
	}

	// TODO: add balance rows

	var buf bytes.Buffer
	for _, row := range rows {
		_, err := buf.Write(row)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (le *LedgerEntry) csvSummaryRow() []byte {
	date := le.DateSubmitted
	year := date.Year()
	period := getFiscalPeriod(int(date.Month()))
	journalDescription := date.Format("January 2006 MAP JE")

	s := fmt.Sprintf(`"1","000000","00001","","GL","JE","%d","%02d",0,"%s","00",0,0,0,2`+"\n",
		year, period, journalDescription)
	return []byte(s)
}

func (le *LedgerEntry) sageTransactionRow(n int, batchDate time.Time) []byte {
	amount := api.Currency(le.Amount)

	s := fmt.Sprintf(
		`"2","000000","00001","%010d","",0,"%s","",%s,"2","%s","%s",%s,"GL","JE",%d,"%s",%d,"%s","%s"`+"\n",
		20*(n+1),
		"19349MMAP12", // TODO: move this to an environment variable
		(-amount).String(),
		le.sageTransactionDescription(),
		le.sageTransactionReference(),
		batchDate.Format("0601")+"01", // can be any date in the batch month

		// TODO: remove the following columns:
		le.RecordType.Int,
		le.EntityCode,
		le.PolicyType.Int,
		le.AccountCostCenter1,
		le.AccountCostCenter2,
	)

	return []byte(s)
}

// TODO: make a better description format unless it has to be the same as before (which I doubt)
func (le *LedgerEntry) sageTransactionDescription() string {
	ccrOrPP := "CCR"
	if le.PolicyType.Valid && le.PolicyType.Int == 2 {
		ccrOrPP = "PP"
	}

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
func (le *LedgerEntry) sageTransactionReference() string {
	// TODO: change this to api.PolicyTypeHousehold (requires a database update)
	if le.EntityCode == "MMB/STM" {
		return fmt.Sprintf("MC %s", le.AccountCostCenter1)
	}
	return le.AccountCostCenter2
}

func getFiscalPeriod(month int) int {
	return (month-domain.Env.FiscalStartMonth+12)%12 + 1
}
