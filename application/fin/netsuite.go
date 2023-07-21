package fin

import (
	"bytes"
	"fmt"

	"github.com/silinternational/cover-api/api"
)

const (
	netSuiteHeader1 = `"RECTYPE","BATCHID", "BTCHENTRY","ORIGCOMP","SRCELEDGER","SRCETYPE","FSCSYR","FSCSPERD","SWEDIT","JRNLDESC","REVPERD","ERRBATCH","ERRENTRY","DETAILCNT","PROCESSCMD"` + "\n"
	netSuiteHeader2 = `"RECTYPE","ACCTID","TRANSAMT","SCURNDEC","TRANSDESC","TRANSREF","TRANSDATE","SRCELDGR","SRCETYPE",` + "\n"

	netSuiteSummaryRowTemplate     = `"1","000000","00001","","GL","JE","%d","%02d",0,"%s","00",0,0,0,2` + "\n"
	netSuiteTransactionRowTemplate = `"2","%s",%s,"2","%s","%s",%s,"GL","JE"` + "\n"
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

func (n *NetSuite) getAccount(t Transaction) string {
	if t.Account != "" {
		return t.Account
	}

	if t.PolicyType == api.PolicyTypeHousehold {
		return t.HouseholdID
	}

	return t.EntityCode
}

func (n *NetSuite) getDescription(t Transaction) string {
	return t.Description
}

func (n *NetSuite) getReference(t Transaction) string {
	if t.Reference != nil {
		return *t.Reference
	}

	// For household policies
	if t.PolicyType == api.PolicyTypeHousehold {
		ref := "MC"

		if t.Name == "" {
			return ref
		}

		return fmt.Sprintf("%s / %s", ref, t.Name)
	}

	// For non-household policies
	ref := fmt.Sprintf("%s%s", t.AccountNumber, t.CostCenter)

	if t.PolicyName == "" {
		return ref
	}

	return fmt.Sprintf("%s / %s", ref, t.PolicyName)
}

func (n *NetSuite) summaryRow() []byte {
	str := fmt.Sprintf(netSuiteSummaryRowTemplate, n.Year, n.Period, n.JournalDescription)
	return []byte(str)
}

func (n *NetSuite) transactionRow(rowNumber int) []byte {
	t := n.Transactions[rowNumber]
	str := fmt.Sprintf(
		netSuiteTransactionRowTemplate,
		n.getAccount(t),
		api.Currency(-t.Amount).String(),
		fmt.Sprintf("%.60s", n.getDescription(t)), // TODO does NetSuite have a 60 char field limit?
		n.getReference(t),
		t.Date.Format("20060102"),
	)
	return []byte(str)
}
