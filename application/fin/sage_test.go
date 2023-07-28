package fin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func TestSage_BatchToCSV(t *testing.T) {
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

	s := &Sage{
		Period:             9,
		Year:               2020,
		JournalDescription: "journal description",
		Transactions:       []Transaction{transaction},
	}

	summaryRow := fmt.Sprintf(`"1","000000","00001","","GL","JE","%d","%02d",0,"%s","00",0,0,0,2`+"\n",
		s.Year, s.Period, s.JournalDescription)

	transactionRow := fmt.Sprintf(`"2","000000","00001","0000000020","",0,"%s","",%s,"2","%s","%s",%s,"GL","JE"`+"\n",
		domain.Env.ExpenseAccount,
		api.Currency(transaction.Amount).String(),
		transaction.Description,
		fmt.Sprintf("MC %s / %s", transaction.HouseholdID, transaction.Name),
		transaction.Date.Format("20060102"),
	)

	want := sageHeader1 + sageHeader2 + summaryRow + transactionRow

	got := s.ToCSV()

	require.Equal(t, want, string(got))
}
