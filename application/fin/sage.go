package fin

import (
	"bytes"
	"fmt"

	"github.com/silinternational/cover-api/api"
)

const (
	sageHeader1 = `"RECTYPE","BATCHID","BTCHENTRY","ORIGCOMP","SRCELEDGER","SRCETYPE","FSCSYR","FSCSPERD","SWEDIT",` +
		`"JRNLDESC","REVPERD","ERRBATCH","ERRENTRY","DETAILCNT","PROCESSCMD"` + "\n"
	sageHeader2 = `"RECTYPE","BATCHNBR","JOURNALID","TRANSNBR","DESCOMP","ROUTE","ACCTID","COMPANYID","TRANSAMT",` +
		`"SCURNDEC","TRANSDESC","TRANSREF","TRANSDATE","SRCELDGR","SRCETYPE",` + "\n"

	sageTransactionRowTemplate = `"2","000000","00001","%010d","",0,"%s","",%s,"2","%s","%s",%s,"GL","JE"` + "\n"
	sageSummaryRowTemplate     = `"1","000000","00001","","GL","JE","%d","%02d",0,"%s","00",0,0,0,2` + "\n"
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

func (s *Sage) BatchToCSV() []byte {
	var buf bytes.Buffer
	buf.Write([]byte(sageHeader1))
	buf.Write([]byte(sageHeader2))
	buf.Write(s.summaryRow())
	for i := range s.Transactions {
		buf.Write(s.transactionRow(i))
	}

	return buf.Bytes()
}

func (s *Sage) summaryRow() []byte {
	str := fmt.Sprintf(sageSummaryRowTemplate, s.Year, s.Period, s.JournalDescription)
	return []byte(str)
}

func (s *Sage) transactionRow(rowNumber int) []byte {
	t := s.Transactions[rowNumber]
	str := fmt.Sprintf(
		sageTransactionRowTemplate,
		20*(rowNumber+1),
		t.Account,
		api.Currency(-t.Amount).String(),
		fmt.Sprintf("%.60s", t.Description), // truncate to Sage limit of 60 characters
		t.Reference,
		t.Date.Format("20060102"),
	)
	return []byte(str)
}
