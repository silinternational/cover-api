package fin

import (
	"bytes"
	"fmt"

	"github.com/silinternational/cover-api/api"
)

const (
	netSuiteHeader1 = `"RECTYPE","BATCHID", "BTCHENTRY","ORIGCOMP","SRCELEDGER","SRCETYPE","FSCSYR","FSCSPERD","SWEDIT","JRNLDESC","REVPERD","ERRBATCH","ERRENTRY","DETAILCNT","PROCESSCMD"` + "\n"
	netSuiteHeader2 = `"RECTYPE","ACCTID","TRANSAMT","SCURNDEC","TRANSDESC","TRANSDATE","SRCELDGR","SRCETYPE",` + "\n"

	netSuiteSummaryRowTemplate     = `"1","000000","00001","","GL","JE","%d","%02d",0,"%s","00",0,0,0,2` + "\n"
	netSuiteTransactionRowTemplate = `"2","%s",%s,"2","%s",%s,"GL","JE"` + "\n"
)

type NetSuite struct {
	Period             int
	Year               int
	JournalDescription string
	Transactions       []Transaction
}

func (n *NetSuite) AppendToBatch(t Transaction) {
	if t.Amount != 0 {
		n.Transactions = append(n.Transactions, t)
	}
}

func (n *NetSuite) BatchToCSV() []byte {
	var buf bytes.Buffer
	buf.Write([]byte(netSuiteHeader1))
	buf.Write([]byte(netSuiteHeader2))
	buf.Write(n.summaryRow())
	for i := range n.Transactions {
		buf.Write(n.transactionRow(i))
	}

	return buf.Bytes()
}

func (n *NetSuite) summaryRow() []byte {
	str := fmt.Sprintf(netSuiteSummaryRowTemplate, n.Year, n.Period, n.JournalDescription)
	return []byte(str)
}

func (n *NetSuite) transactionRow(rowNumber int) []byte {
	t := n.Transactions[rowNumber]
	str := fmt.Sprintf(
		netSuiteTransactionRowTemplate,
		t.Account,
		api.Currency(-t.Amount).String(),
		fmt.Sprintf("%.60s", t.Description), // TODO does NetSuite have a 60 char field limit?
		t.Date.Format("20060102"),
	)
	return []byte(str)
}
