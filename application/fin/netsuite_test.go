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
		Account:     "xyz123",
		Amount:      0,
		Description: "transaction description",
		Reference:   "abc123",
		Date:        time.Now(),
	}

	s := &NetSuite{
		Period:             9,
		Year:               2020,
		JournalDescription: "journal description",
		Transactions:       []Transaction{transaction},
	}

	summaryRow := fmt.Sprintf(`"1","000000","00001","","GL","JE","%d","%02d",0,"%s","00",0,0,0,2`+"\n",
		s.Year, s.Period, s.JournalDescription)

	transactionRow := fmt.Sprintf(`"2","%s",%s,"2","%s",%s,"GL","JE"`+"\n",
		transaction.Account,
		api.Currency(transaction.Amount).String(),
		transaction.Description,
		transaction.Date.Format("20060102"),
	)

	want := netSuiteHeader1 + netSuiteHeader2 + summaryRow + transactionRow

	got := s.BatchToCSV()

	require.Equal(t, want, string(got))
}
