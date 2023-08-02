package fin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func TestPolicy_Export(t *testing.T) {
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

	p := &Policy{
		Transactions: []Transaction{transaction},
	}

	transactionRow := fmt.Sprintf(policyTransactionRowTemplate,
		api.Currency(-transaction.Amount),
		transaction.Description,
		fmt.Sprintf("MC %s / %s", transaction.HouseholdID, transaction.Name),
		transaction.Date.Format(domain.DateFormat),
	)

	want := policyHeader + transactionRow

	got, gotType := p.RenderBatch()

	require.Equal(t, want, string(got))
	require.Equal(t, domain.ContentCSV, gotType)
}
