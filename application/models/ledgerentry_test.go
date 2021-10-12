package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

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
		ms.NoError(f.Items[i].Approve(ms.DB, false))

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
		PolicyID:         domain.GetUUID(),
		EntityCode:       "EntityCode",
		RiskCategoryName: "Mobile",
		Type:             LedgerEntryTypeClaim,
		FirstName:        "FirstName",
		LastName:         "LastName",
		Amount:           100,
		DateSubmitted:    date,
		AccountNumber:    "AccountNumber",
		HouseholdID:      "HouseholdID",
		CostCenter:       "CostCenter",
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
					entry.getIncomeAccount(),
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
		{PolicyID: policy1, EntityCode: "ABC", RiskCategoryCC: "12345"},
		{PolicyID: policy2, EntityCode: "", RiskCategoryCC: "67890"},
		{PolicyID: policy3, EntityCode: "", RiskCategoryCC: "67890"},
	}
	blocks := entries.MakeBlocks()
	ms.Equal(2, len(blocks))

	ms.Equal(1, len(blocks["4425012345"]))
	ms.Equal(policy1, blocks["4425012345"][0].PolicyID)

	ms.Equal(2, len(blocks["4020067890"]))
	ms.Equal(policy2, blocks["4020067890"][0].PolicyID)
	ms.Equal(policy3, blocks["4020067890"][1].PolicyID)
}

func (ms *ModelSuite) Test_NewLedgerEntry() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 2, ClaimsPerPolicy: 1})
	householdPolicy := f.Policies[0]
	householdPolicyItem := householdPolicy.Items[0]
	ms.NoError(householdPolicyItem.setAccountablePerson(ms.DB, f.Users[0].ID))
	householdPolicyClaim := f.Policies[0].Claims[0]
	ms.False(uuid.Nil == householdPolicyClaim.ID, "householdPolicyClaim is not hydrated")

	corporatePolicy := ConvertPolicyType(ms.DB, f.Policies[1])
	corporatePolicyItem := corporatePolicy.Items[0]
	ms.NoError(corporatePolicyItem.setAccountablePerson(ms.DB, f.Users[1].ID))

	tests := []struct {
		name   string
		policy Policy
		item   *Item
		claim  *Claim
	}{
		{
			name:   "household policy item with claim",
			policy: householdPolicy,
			item:   &householdPolicyItem,
			claim:  &householdPolicyClaim,
		},
		{
			name:   "corporate policy item no claim",
			policy: corporatePolicy,
			item:   &corporatePolicyItem,
		},
		{
			name:   "policy only",
			policy: corporatePolicy,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			le := NewLedgerEntry(tt.policy, tt.item, tt.claim)

			ms.Equal(tt.policy.ID, le.PolicyID, "PolicyID is incorrect")
			ms.WithinDuration(time.Now().UTC(), le.DateSubmitted, time.Minute, "DateSubmitted is incorrect")
			ms.Equal(tt.policy.Type, le.PolicyType, "PolicyType is incorrect")
			if tt.policy.Type == api.PolicyTypeCorporate {
				ms.Equal(tt.policy.Account, le.AccountNumber, "AccountNumber is incorrect")
				ms.Equal(tt.policy.CostCenter+" / "+tt.policy.AccountDetail, le.CostCenter, "CostCenter is incorrect")
				ms.Equal(tt.policy.EntityCode.Code, le.EntityCode, "EntityCode is incorrect")
			} else {
				ms.Equal(tt.policy.HouseholdID.String, le.HouseholdID, "HouseholdID is incorrect")
			}

			if tt.item == nil {
				ms.False(le.ItemID.Valid, "ItemID is not nil")
			} else {
				ms.Equal(tt.item.RiskCategory.Name, le.RiskCategoryName, "RiskCategoryName is incorrect")
				ms.Equal(tt.item.RiskCategory.CostCenter, le.RiskCategoryCC, "RiskCategoryCC is incorrect")
				ms.Equal(tt.item.ID, le.ItemID.UUID, "ItemID is incorrect")
			}

			if tt.claim == nil {
				ms.False(le.ClaimID.Valid, "ClaimID is not nil")
			} else {
				ms.Equal(tt.claim.ID, le.ClaimID.UUID, "ClaimID is incorrect")
			}
		})
	}
}
