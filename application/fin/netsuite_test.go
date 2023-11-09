package fin

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
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

	n := newNetSuiteReport("journal description", "", now)
	n.AppendToBatch("", t1)
	n.AppendToBatch("bar", t2)

	transaction1Row := fmt.Sprintf(netSuiteTransactionRowTemplate,
		"MAP",                             // SystemSubsidiary
		"",                                // GroupID, left blank
		n.rowID+1,                         // TransactionID
		t1.Date.Format("01/02/2006"),      // TransactionDate
		t1.Description,                    // Description
		n.getDebitAccount(t1),             // DebitAccount
		n.getCreditAccount(t1),            // CreditAccount
		"",                                // InterCoAccount, left blank
		api.Currency(-t1.Amount).String(), // Amount
		"USD",                             // Currency
		n.getReference(t1),
	)

	transaction2Row := fmt.Sprintf(netSuiteTransactionRowTemplate,
		"MAP",                             // SystemSubsidiary
		"",                                // GroupID, left blank
		n.rowID+2,                         // TransactionID
		t2.Date.Format("01/02/2006"),      // TransactionDate
		t2.Description,                    // Description
		n.getDebitAccount(t2),             // DebitAccount
		n.getCreditAccount(t2),            // CreditAccount
		"",                                // InterCoAccount, left blank
		api.Currency(-t2.Amount).String(), // Amount
		"USD",                             // Currency
		n.getReference(t2),
	)

	got, gotType := n.RenderBatch()
	require.Equal(t, domain.ContentZip, gotType)

	reader := bytes.NewReader(got)
	r, err := zip.NewReader(reader, int64(len(got)))
	require.NoError(t, err)

	files := r.File
	require.Equal(t, 2, len(files))

	for _, f := range files {
		date := n.date.Format(domain.DateFormat)
		name := f.Name[:len(f.Name)-4]
		require.True(t, strings.HasSuffix(name, date))

		name = name[:len(name)-len(date)-1]
		require.Contains(t, n.TransactionBlocks, name)

		contents, err := f.Open()
		require.NoError(t, err)

		body, err := io.ReadAll(contents)
		require.NoError(t, err)

		row := transaction1Row
		if name == "bar" {
			row = transaction2Row
		}

		// don't try to compare the row number since we can't guarantee the transaction batch ordering
		_, want, _ := strings.Cut(row, ",")

		require.Contains(t, string(body), want)
	}
}
