package fin

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"

	"github.com/silinternational/cover-api/api"
)

var (
	header1 = []string{"RECTYPE", "BATCHID", "BTCHENTRY", "ORIGCOMP", "SRCELEDGER", "SRCETYPE", "FSCSYR", "FSCSPERD", "SWEDIT",
		"JRNLDESC", "REVPERD", "ERRBATCH", "ERRENTRY", "DETAILCNT", "PROCESSCMD"}
	header2 = []string{"RECTYPE", "BATCHNBR", "JOURNALID", "TRANSNBR", "DESCOMP", "ROUTE", "ACCTID", "COMPANYID", "TRANSAMT",
		"SCURNDEC", "TRANSDESC", "TRANSREF", "TRANSDATE", "SRCELDGR", "SRCETYPE"}
)

type Sage struct {
	Period             int
	Year               int
	JournalDescription string
	Transactions       []Transaction
}

func (s *Sage) AppendToBatch(t Transaction) {
	if t.Amount != 0 {
		s.Transactions = append(s.Transactions, t)
	}
}

func (s *Sage) BatchToCSV() ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	defer writer.Flush()

	for _, s := range [][]string{header1, header2, s.summaryRow()} {
		if err := writer.Write(s); err != nil {
			return []byte{}, err
		}
	}

	for i := range s.Transactions {
		if err := writer.Write(s.transactionRow(i)); err != nil {
			return []byte{}, errors.New("error writing sage batch to csv: " + err.Error())
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return nil, errors.New("error closing sage csv output: " + err.Error())
	}

	return buf.Bytes(), nil
}

func (s *Sage) summaryRow() []string {
	// `"1","000000","00001","","GL","JE","%d","%02d",0,"%s","00",0,0,0,2`
	return []string{
		"1",
		"000000",
		"00001",
		"",
		"GL",
		"JE",
		fmt.Sprintf("%d", s.Year),
		fmt.Sprintf("%02d", s.Period),
		"0",
		s.JournalDescription,
		"00",
		"0",
		"0",
		"0",
		"2",
	}
}

func (s *Sage) transactionRow(rowNumber int) []string {
	t := s.Transactions[rowNumber]

	// `"2","000000","00001","%010d","",0,"%s","",%s,"2","%s","%s",%s,"GL","JE"`
	return []string{
		"2",
		"000000",
		"00001",
		fmt.Sprintf("%010d", 20*(rowNumber+1)),
		"",
		"0",
		t.Account,
		"",
		api.Currency(-t.Amount).String(),
		"2",
		fmt.Sprintf("%.60s", t.Description),
		t.Reference,
		t.Date.Format("20060102"),
		"GL",
		"JE",
	}
}
