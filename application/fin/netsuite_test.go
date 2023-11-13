package fin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func TestNetSuite_Export(t *testing.T) {
	now := time.Now().UTC()
	t1 := Transaction{
		EntityCode:        "abc1",
		RiskCategoryName:  "def2",
		RiskCategoryCC:    "ghi3",
		Type:              "jkl4",
		PolicyType:        api.PolicyTypeHousehold,
		HouseholdID:       "mno5",
		IncomeAccount:     "pqr6",
		Name:              "stu7",
		ClaimPayoutOption: "vwx8",
		Amount:            1,
		Date:              now,
		Description:       "transaction description",
	}
	t2 := Transaction{
		EntityCode:        "zyx9",
		RiskCategoryName:  "wvu8",
		RiskCategoryCC:    "tsr7",
		Type:              "qpo6",
		PolicyType:        api.PolicyTypeTeam,
		PolicyName:        "nml5",
		AccountNumber:     "kji4",
		CostCenter:        "hgf3",
		ClaimPayoutOption: "edc2",
		Amount:            2,
		Date:              now,
		Description:       "transaction description",
	}
	s1 := Transaction{Account: "summaryONE", Amount: t1.Amount}
	s2 := Transaction{Account: "summaryTWO", Amount: t2.Amount}

	n := newNetSuiteReport("journal description", "", now)
	n.AppendToBatch("", t1)
	n.AppendToBatch("", s1)
	n.AppendToBatch("bar", t2)
	n.AppendToBatch("bar", s2)

	transaction1Row := fmt.Sprintf(netSuiteTransactionRowTemplate,
		n.rowID+1,                         // TransactionID
		t1.Date.Format("01/02/2006"),      // TransactionDate
		t1.Description,                    // Description
		n.getDebitAccount(t1),             // DebitAccount
		s1.Account,                        // CreditAccount
		t1.CostCenter,                     // InterCoAccount
		api.Currency(-t1.Amount).String(), // Amount
		n.getReference(t1),
	)

	transaction2Row := fmt.Sprintf(netSuiteTransactionRowTemplate,
		n.rowID+2,                         // TransactionID
		t2.Date.Format("01/02/2006"),      // TransactionDate
		t2.Description,                    // Description
		n.getDebitAccount(t2),             // DebitAccount
		s2.Account,                        // CreditAccount
		t2.CostCenter,                     // InterCoAccount
		api.Currency(-t2.Amount).String(), // Amount
		n.getReference(t2),
	)

	want := netSuiteHeader + transaction1Row + transaction2Row

	got, gotType := n.RenderBatch()
	require.Equal(t, want, string(got))
	require.Equal(t, domain.ContentCSV, gotType)
}
