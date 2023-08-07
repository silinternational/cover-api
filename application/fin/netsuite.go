package fin

import (
	"archive/zip"
	"bytes"
	"fmt"
	"log"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

const (
	netSuiteHeader                 = `"TRANSID","ACCTID","TRANSAMT","TRANSDESC","TRANSREF","TRANSDATE"` + "\n"
	netSuiteTransactionRowTemplate = `%s,"%s",%s,"%s","%s",%s` + "\n"
)

var netsuiteRowNum int

type NetSuite struct {
	Period             int
	Year               int
	JournalDescription string
	TransactionBlocks  TransactionBlocks
}

func (n *NetSuite) AppendToBatch(block string, t Transaction) {
	if t.Amount != 0 {
		n.TransactionBlocks[block] = append(n.TransactionBlocks[block], t)
	}
}

func (n *NetSuite) RenderBatch() ([]byte, string) {
	// Create a buffer to write our archive to.
	buff := new(bytes.Buffer)

	// Create a new zip archive.
	w := zip.NewWriter(buff)

	for blockName, block := range n.TransactionBlocks {
		f, err := w.Create(blockName + ".csv")
		if err != nil {
			log.Fatal(err)
		}

		contents := n.generateCSV(block)
		if _, err = f.Write(contents); err != nil {
			log.Fatal(err)
		}
	}

	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	return buff.Bytes(), domain.ContentZip
}

func (n *NetSuite) generateCSV(transactions Transactions) []byte {
	var buf bytes.Buffer
	buf.Write([]byte(netSuiteHeader))
	for _, transaction := range transactions {
		buf.Write(n.transactionRow(transaction))
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

func (n *NetSuite) getRowID() string {
	return fmt.Sprintf("%d%02d%06d", n.Year, n.Period, netsuiteRowNum)
}

func (n *NetSuite) transactionRow(t Transaction) []byte {
	netsuiteRowNum++
	str := fmt.Sprintf(
		netSuiteTransactionRowTemplate,
		n.getRowID(),
		n.getAccount(t),
		api.Currency(-t.Amount).String(),
		t.Description,
		n.getReference(t),
		t.Date.Format("20060102"),
	)
	return []byte(str)
}
