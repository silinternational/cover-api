package models

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo"
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

func LatestBatch(c buffalo.Context) ([]byte, error) {
	tx := Tx(c)

	var le LedgerEntries
	if err := le.FindLatestBatch(tx); err != nil {
		return nil, err
	}

	return le.ToCsv()
}

func BeginningOfLastMonth(date time.Time) time.Time {
	return date.AddDate(0, -1, -date.Day()+1)
}

func EndOfLastMonth(date time.Time) time.Time {
	return date.AddDate(0, 0, -date.Day())
}

func (le *LedgerEntries) FindLatestBatch(tx *pop.Connection) error {
	now := time.Date(2021, 07, 01, 0, 0, 0, 0, time.UTC)
	// now := time.Now().UTC()
	today := now.Truncate(time.Hour * 24)
	firstDay := BeginningOfLastMonth(today)
	lastDay := EndOfLastMonth(today)

	err := tx.Where("date_submitted BETWEEN ? and ?", firstDay, lastDay).
		// Where("date_entered IS NULL").
		All(le)

	return appErrorFromDB(err, api.ErrorQueryFailure)
}

func (le *LedgerEntries) ToCsv() ([]byte, error) {
	const header1 = `"RECTYPE","BATCHID","BTCHENTRY","ORIGCOMP","SRCELEDGER","SRCETYPE","FSCSYR","FSCSPERD","SWEDIT","JRNLDESC","REVPERD","ERRBATCH","ERRENTRY","DETAILCNT","PROCESSCMD"` + "\n"
	const header2 = `"RECTYPE","BATCHNBR","JOURNALID","TRANSNBR","DESCOMP","ROUTE","ACCTID","COMPANYID","TRANSAMT","SCURNDEC","TRANSDESC","TRANSREF","TRANSDATE","SRCELDGR","SRCETYPE"` + "\n"

	rows := [][]byte{[]byte(header1), []byte(header2)}

	if len(*le) == 0 {
		return nil, errors.New("no ledger entries, cannot convert to CSV")
	}

	rows = append(rows, (*le)[0].csvSummaryRow())

	for i, l := range *le {
		newRow := l.sageTransactionRow(i)
		if strings.Contains(string(newRow), "not impl") {
			continue
		}
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
	period := getPeriod(int(date.Month()))
	journalDescription := date.Format("January 2006 MAP JE")

	s := fmt.Sprintf(`"1","000000","00001","","GL","JE","%d","%02d",0,"%s","00",0,0,0,2`+"\n",
		year, period, journalDescription)
	return []byte(s)
}

func (le *LedgerEntry) sageTransactionRow(n int) []byte {
	amount := api.Currency(le.Amount)

	s := fmt.Sprintf(
		`"2","000000","00001","%010d","",0,"%s","",%s,"2","%s","%s",%d,"GL","JE"`+"\n",
		20*(n+1),
		le.sageAccount(),
		-amount,
		le.sageTransactionDescription(),
		"MC 242769",
		20210615,
	)

	return []byte(s)
}

func (le *LedgerEntry) sageTransactionDescription() string {
	switch le.RecordType.Int { // TODO: change RecordType column to string
	case 3: // ?
		return "3"
	case 4: // Claim
		// TODO: figure out why names are blank in legacy data AND ensure they are not blank in new data
		dateString := le.DateSubmitted.Format("Jan 02, 2006")
		return fmt.Sprintf("%s,%s CCR Claim %s", le.LastName, le.FirstName, dateString)
	}
	return "(RecordType not impl)"
}

func (le *LedgerEntry) sageAccount() string {
	switch le.RecordType.Int {
	case 3: // ?
		return "3"
	case 4: // Claim
		return "19340MMAP12"
	}
	return "(AccountID not impl)"
}

func getPeriod(month int) int {
	return (month-domain.Env.FiscalStartMonth+12)%12 + 1
}
