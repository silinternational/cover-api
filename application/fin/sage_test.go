package fin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/silinternational/cover-api/api"
)

func TestSage_BatchToCSV(t *testing.T) {
	transaction := Transaction{
		Account:     "xyz123",
		Amount:      0,
		Description: "transaction description",
		Reference:   "abc123",
		Date:        time.Now(),
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
		transaction.Account,
		api.Currency(transaction.Amount).String(),
		transaction.Description,
		transaction.Reference,
		transaction.Date.Format("20060102"),
	)

	want := header1 + header2 + summaryRow + transactionRow

	got := s.BatchToCSV()

	require.Equal(t, want, string(got))
}
