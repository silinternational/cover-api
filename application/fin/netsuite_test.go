package fin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/silinternational/cover-api/api"
)

func TestNetSuite_BatchToCSV(t *testing.T) {
	transaction := Transaction{
		EntityCode:        "abc1",
		RiskCategoryName:  "def2",
		RiskCategoryCC:    "ghi3",
		Type:              "jkl4",
		PolicyType:        api.PolicyTypeHousehold,
		HouseholdID:       "mno5",
		IncomeAccount:     "pqr6",
		Name:              "stu7",
		ClaimPayoutOption: "vwx8",
		Amount:            0,
		Date:              time.Now(),
		Description:       "transaction description",
	}

	n := &NetSuite{
		Period:             9,
		Year:               2020,
		JournalDescription: "journal description",
		Transactions:       []Transaction{transaction},
	}

	summaryRow := fmt.Sprintf(`"1","000000","00001","","GL","JE","%d","%02d",0,"%s","00",0,0,0,2`+"\n",
		n.Year, n.Period, n.JournalDescription)

	transactionRow := fmt.Sprintf(netSuiteTransactionRowTemplate,
		transaction.HouseholdID,
		api.Currency(transaction.Amount).String(),
		transaction.Description,
		fmt.Sprintf("MC / %s", transaction.Name),
		transaction.Date.Format("20060102"),
	)

	want := netSuiteHeader1 + netSuiteHeader2 + summaryRow + transactionRow

	got := n.BatchToCSV()

	require.Equal(t, want, string(got))
}
