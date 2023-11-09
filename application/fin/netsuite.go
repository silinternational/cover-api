package fin

import (
	"archive/zip"
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

const (
	netSuiteHeader                 = `"SystemSubsidiary","GroupID","TransactionID","TransactionDate","Description","DebitAccount","CreditAccount","InterCoAccount","Amount","Currency","Reference"\n`
	netSuiteTransactionRowTemplate = `USA,,%d,%s,"%s","%s","%s",,%s,USD,"%s"` + "\n"
)

type NetSuite struct {
	Period             int
	Year               int
	JournalDescription string
	TransactionBlocks  TransactionBlocks

	date  time.Time
	rowID int64
}

func newNetSuiteReport(batchDesc, reportType string, date time.Time) *NetSuite {
	period := getFiscalPeriod(int(date.Month()))
	year := getFiscalYear(date)
	rowID := int64(((year * 100) + period) * 10)
	if reportType == "Annual" {
		rowID++
	}
	rowID *= 100000

	return &NetSuite{
		Period:             period,
		Year:               year,
		JournalDescription: batchDesc,
		TransactionBlocks:  make(TransactionBlocks),
		date:               date,
		rowID:              rowID,
	}
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
		f, err := w.Create(fmt.Sprintf("%s_%s.csv", blockName, n.date.Format(domain.DateFormat)))
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

func (n *NetSuite) getCreditAccount(t Transaction) string {
	return fmt.Sprintf("%s%s", t.AccountNumber, t.CostCenter)
}

func (n *NetSuite) getDebitAccount(t Transaction) string {
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
		return t.Name
	}

	// For non-household policies
	return t.PolicyName
}

func (n *NetSuite) transactionRow(t Transaction) []byte {
	n.rowID++

	// Prefix reference with transaction ID, which isn't available until now
	ref := n.getReference(t)
	if ref != "" {
		ref = fmt.Sprintf("%d / %s", n.rowID, ref)
	}

	str := fmt.Sprintf(
		netSuiteTransactionRowTemplate,
		n.rowID,                          // TransactionID
		t.Date.Format("01/02/2006"),      // TransactionDate
		t.Description,                    // Description
		n.getDebitAccount(t),             // DebitAccount
		n.getCreditAccount(t),            // CreditAccount
		api.Currency(-t.Amount).String(), // Amount
		ref,                              // Reference
	)
	return []byte(str)
}
