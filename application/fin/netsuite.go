package fin

import (
	"bytes"
	"fmt"
	"time"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

const (
	netSuiteHeader                 = `"SystemSubsidiary","GroupID","TransactionID","TransactionDate","Description","DebitAccount","CreditAccount","InterCoAccount","Amount","Currency","Reference","ExchangeRate"` + "\n"
	netSuiteTransactionRowTemplate = `MAP,,%d,%s,"%s","%s","%s",%s,%s,USD,"%s",` + "\n"
)

type NetSuite struct {
	Period             int
	Year               int
	JournalDescription string
	TransactionBlocks  TransactionBlocks

	date       time.Time
	rowID      int64
	blockNames []string
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
		blockNames:         []string{},
	}
}

func (n *NetSuite) AppendToBatch(block string, t Transaction) {
	if t.Amount == 0 {
		return
	}

	if _, ok := n.TransactionBlocks[block]; !ok {
		n.blockNames = append(n.blockNames, block)
	}

	n.TransactionBlocks[block] = append(n.TransactionBlocks[block], t)
}

func (n *NetSuite) RenderBatch() ([]byte, string) {
	var buf bytes.Buffer
	buf.Write([]byte(netSuiteHeader))

	for _, name := range n.blockNames {
		block := n.TransactionBlocks[name]
		last := len(block) - 1

		creditAccount := block[last].Account

		// TODO clean this up when Sage report is removed
		// Last row is a summary row. Can't remove it completely
		// as it is being used by Sage, so removing it here
		for _, transaction := range block[:last] {
			buf.Write(n.transactionRow(transaction, creditAccount))
		}
	}

	return buf.Bytes(), domain.ContentCSV
}

func (n *NetSuite) getDebitAccount(t Transaction) string {
	if t.Account != "" {
		return t.Account
	}

	if t.PolicyType == api.PolicyTypeHousehold {
		return t.HouseholdID + "_C"
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

func (n *NetSuite) transactionRow(t Transaction, creditAccount string) []byte {
	n.rowID++

	str := fmt.Sprintf(
		netSuiteTransactionRowTemplate,
		n.rowID,                          // TransactionID
		t.Date.Format("01/02/2006"),      // TransactionDate
		t.Description,                    // Description
		n.getDebitAccount(t),             // DebitAccount
		creditAccount,                    // CreditAccount
		t.CostCenter,                     // InterCoAccount
		api.Currency(-t.Amount).String(), // Amount
		n.getReference(t),                // Reference
	)
	return []byte(str)
}
