package fin

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func TestNetSuite_Export(t *testing.T) {
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
		Amount:            0,
		Date:              time.Now(),
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
		Amount:            0,
		Date:              time.Now(),
		Description:       "transaction description",
	}

	n := &NetSuite{
		Period:             9,
		Year:               2020,
		JournalDescription: "journal description",
		TransactionBlocks:  TransactionBlocks{"": Transactions{t1}, "bar": Transactions{t2}},
	}

	transaction1Row := fmt.Sprintf(netSuiteTransactionRowTemplate,
		fmt.Sprintf("%d%02d%06d", n.Year, n.Period, 1),
		n.getAccount(t1),
		api.Currency(t1.Amount).String(),
		t1.Description,
		n.getReference(t1),
		t1.Date.Format("20060102"),
	)

	transaction2Row := fmt.Sprintf(netSuiteTransactionRowTemplate,
		fmt.Sprintf("%d%02d%06d", n.Year, n.Period, 2),
		n.getAccount(t2),
		api.Currency(t2.Amount).String(),
		t2.Description,
		n.getReference(t2),
		t2.Date.Format("20060102"),
	)

	got, gotType := n.RenderBatch()
	require.Equal(t, domain.ContentZip, gotType)

	reader := bytes.NewReader(got)
	r, err := zip.NewReader(reader, int64(len(got)))
	require.NoError(t, err)

	files := r.File
	require.Equal(t, 2, len(files))

	for _, f := range files {
		name := f.Name[:len(f.Name)-4]
		require.Contains(t, n.TransactionBlocks, name)

		contents, err := f.Open()
		require.NoError(t, err)

		body, err := io.ReadAll(contents)
		require.NoError(t, err)

		want := transaction1Row
		if name == "bar" {
			want = transaction2Row
		}

		require.Equal(t, netSuiteHeader+want, string(body))
	}
}
