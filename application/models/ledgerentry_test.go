package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestLedgerEntries_AllForMonth() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2})

	march := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	april := time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC)
	may := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)

	datesSubmitted := []time.Time{march, april}
	datesEntered := []nulls.Time{nulls.NewTime(april), {}}

	for i := range f.Items {
		ms.NoError(f.Items[i].Approve(ms.DB))

		entry := LedgerEntry{}
		ms.NoError(ms.DB.Where("item_id = ?", f.Items[i].ID).First(&entry))
		entry.DateSubmitted = datesSubmitted[i]
		entry.DateEntered = datesEntered[i]
		ms.NoError(ms.DB.Update(&entry))
	}

	tests := []struct {
		name                    string
		batchDate               time.Time
		expectedNumberOfEntries int
	}{
		{
			name:                    "no un-entered entries for March",
			batchDate:               march,
			expectedNumberOfEntries: 0,
		},
		{
			name:                    "one entry for April",
			batchDate:               april,
			expectedNumberOfEntries: 1,
		},
		{
			name:                    "no entry for May",
			batchDate:               may,
			expectedNumberOfEntries: 0,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			entries := LedgerEntries{}
			err := entries.AllForMonth(ms.DB, tt.batchDate)
			ms.NoError(err)
			ms.Equal(tt.expectedNumberOfEntries, len(entries), "incorrect number of LedgerEntries")
		})
	}
}

func (ms *ModelSuite) TestLedgerEntries_ToCsv() {
	date := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)

	entry := LedgerEntry{
		PolicyID:           domain.GetUUID(),
		EntityCode:         "EntityCode",
		RiskCategoryName:   "Mobile",
		Type:               LedgerEntryTypeClaim,
		IncomeAccount:      "IncomeAccount",
		FirstName:          "FirstName",
		LastName:           "LastName",
		Amount:             100,
		DateSubmitted:      date,
		AccountNumber:      "AccountNumber",
		AccountCostCenter1: "AccountCostCenter1",
		AccountCostCenter2: "AccountCostCenter2",
	}

	domain.Env.ExpenseAccount = "XYZ123"

	summaryLine := date.Format("January 2006 Cover JE")

	tests := []struct {
		name      string
		entries   LedgerEntries
		batchDate time.Time
		want      []string
	}{
		{
			name:      "no data",
			entries:   LedgerEntries{},
			batchDate: date,
			want:      []string{summaryLine},
		},
		{
			name:      "1 entry",
			entries:   LedgerEntries{entry},
			batchDate: date,
			want: []string{
				summaryLine,
				fmt.Sprintf(`"2","000000","00001","0000000020","",0,"%s","",%s,"2","%s","%s",%s,"GL","JE"`,
					domain.Env.ExpenseAccount,
					api.Currency(-entry.Amount).String(),
					entry.transactionDescription(),
					entry.transactionReference(),
					date.Format("20060102"),
				),
				fmt.Sprintf(`"2","000000","00001","0000000040","",0,"%s","",%s,"2","%s","",%s,"GL","JE"`,
					entry.IncomeAccount,
					api.Currency(entry.Amount).String(),
					entry.balanceDescription(),
					date.Format("20060102"),
				),
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.entries.ToCsv(tt.batchDate)
			for _, w := range tt.want {
				ms.Contains(string(got), w)
			}
		})
	}
}

func (ms *ModelSuite) TestLedgerEntries_MakeBlocks() {
	policy1 := domain.GetUUID()
	policy2 := domain.GetUUID()
	policy3 := domain.GetUUID()

	entries := LedgerEntries{
		{PolicyID: policy1, IncomeAccount: "1"},
		{PolicyID: policy2, IncomeAccount: "2"},
		{PolicyID: policy3, IncomeAccount: "2"},
	}
	blocks := entries.MakeBlocks()
	ms.Equal(2, len(blocks))

	ms.Equal(1, len(blocks["1"]))
	ms.Equal(policy1, blocks["1"][0].PolicyID)

	ms.Equal(2, len(blocks["2"]))
	ms.Equal(policy2, blocks["2"][0].PolicyID)
	ms.Equal(policy3, blocks["2"][1].PolicyID)
}
