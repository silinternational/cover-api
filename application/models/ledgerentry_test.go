package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/fin"
)

func (ms *ModelSuite) TestLedgerEntries_AllForMonth() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2})
	user := f.Users[0]
	ctx := CreateTestContext(user)

	march := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	april := time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC)
	may := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)

	datesSubmitted := []time.Time{march, april}
	datesEntered := []nulls.Time{nulls.NewTime(april), {}}

	for i := range f.Items {
		ms.NoError(f.Items[i].Approve(ctx, false))

		entry := LedgerEntry{}
		ms.NoError(ms.DB.Where("item_id = ?", f.Items[i].ID).First(&entry))
		entry.DateSubmitted = datesSubmitted[i]
		entry.DateEntered = datesEntered[i]
		ms.NoError(ms.DB.Update(&entry))
	}

	tests := []struct {
		name                    string
		cutoffDate              time.Time
		expectedNumberOfEntries int
	}{
		{
			name:                    "no entries prior to March 1",
			cutoffDate:              march,
			expectedNumberOfEntries: 0,
		},
		{
			name:                    "no un-entered entries prior to April 1",
			cutoffDate:              april,
			expectedNumberOfEntries: 0,
		},
		{
			name:                    "one entry prior to May 1",
			cutoffDate:              may,
			expectedNumberOfEntries: 1,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			entries := LedgerEntries{}
			err := entries.AllNotEntered(ms.DB, tt.cutoffDate)
			ms.NoError(err)
			ms.Equal(tt.expectedNumberOfEntries, len(entries), "incorrect number of LedgerEntries")
		})
	}
}

func (ms *ModelSuite) TestLedgerEntries_ToCsvForPolicy() {
	date := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)

	entry := LedgerEntry{
		PolicyID:         domain.GetUUID(),
		EntityCode:       "EntityCode",
		RiskCategoryName: "Mobile",
		Type:             LedgerEntryTypeClaim,
		Name:             "MyColleague",
		PolicyName:       "OurPolicy",
		Amount:           100,
		DateSubmitted:    date,
		AccountNumber:    "AccountNumber",
		IncomeAccount:    "12345",
		HouseholdID:      "HouseholdID",
		CostCenter:       "CostCenter",
	}

	tests := []struct {
		name    string
		format  string
		entries LedgerEntries
		want    []string
	}{
		{
			name:    "no data",
			format:  fin.ReportFormatPolicy,
			entries: LedgerEntries{},
			want:    []string{csvPolicyHeader},
		},
		{
			name:    "1 entry",
			format:  fin.ReportFormatPolicy,
			entries: LedgerEntries{entry},
			want: []string{
				csvPolicyHeader,
				fmt.Sprintf(`%s,"%s","%s",%s`,
					entry.Amount.String(),
					entry.getDescription(),
					getReference(entry),
					date.Format(domain.DateFormat),
				),
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.entries.ToCsvForPolicy()
			for _, w := range tt.want {
				ms.Contains(string(got), w)
			}
		})
	}
}

func (ms *ModelSuite) TestLedgerEntries_ToCsv() {
	date := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)

	entry := LedgerEntry{
		PolicyID:         domain.GetUUID(),
		PolicyType:       api.PolicyTypeHousehold,
		EntityCode:       "EntityCode",
		RiskCategoryName: "Mobile",
		Type:             LedgerEntryTypeClaim,
		Name:             "MyColleague",
		PolicyName:       "OurPolicy",
		Amount:           100,
		DateSubmitted:    date,
		AccountNumber:    "AccountNumber",
		IncomeAccount:    "12345",
		HouseholdID:      "HouseholdID",
		CostCenter:       "CostCenter",
	}
	teamEntry := LedgerEntry{
		PolicyID:         domain.GetUUID(),
		PolicyType:       api.PolicyTypeTeam,
		EntityCode:       "EntityCode",
		RiskCategoryName: "Mobile",
		Type:             LedgerEntryTypeClaim,
		Name:             "MyColleague",
		PolicyName:       "OurPolicy",
		Amount:           345,
		DateSubmitted:    date,
		AccountNumber:    "AccountNumber",
		IncomeAccount:    "12345",
		HouseholdID:      "Team",
		CostCenter:       "CostCenter",
	}

	domain.Env.ExpenseAccount = "XYZ123"

	summaryLine := fmt.Sprintf("%s %s JE", date.Format("January 2006"), domain.Env.AppName)

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
					entry.getDescription(),
					getReference(entry),
					date.Format("20060102"),
				),
				fmt.Sprintf(`"2","000000","00001","0000000040","",0,"%s","",%s,"2","%s","",%s,"GL","JE"`,
					entry.IncomeAccount+entry.RiskCategoryCC,
					entry.Amount.String(),
					entry.balanceDescription(),
					date.Format("20060102"),
				),
			},
		},
		{
			name:      "split claim entries",
			entries:   LedgerEntries{entry, teamEntry},
			batchDate: date,
			want: []string{
				summaryLine,
				fmt.Sprintf(`"2","000000","00001","0000000020","",0,"%s","",%s,"2","%s","%s",%s,"GL","JE"`,
					domain.Env.ExpenseAccount,
					api.Currency(-entry.Amount).String(),
					entry.getDescription(),
					entry.getReference(),
					date.Format("20060102"),
				),
				fmt.Sprintf(`"2","000000","00001","0000000040","",0,"%s","",%s,"2","%s","",%s,"GL","JE"`,
					entry.IncomeAccount+entry.RiskCategoryCC,
					entry.Amount.String(),
					entry.balanceDescription(),
					date.Format("20060102"),
				),
				fmt.Sprintf(`"2","000000","00001","0000000060","",0,"%s","",%s,"2","%s","%s",%s,"GL","JE"`,
					domain.Env.ExpenseAccount,
					api.Currency(-teamEntry.Amount).String(),
					teamEntry.getDescription(),
					teamEntry.getReference(),
					date.Format("20060102"),
				),
				fmt.Sprintf(`"2","000000","00001","0000000080","",0,"%s","",%s,"2","%s","",%s,"GL","JE"`,
					teamEntry.IncomeAccount+teamEntry.RiskCategoryCC,
					teamEntry.Amount.String(),
					teamEntry.balanceDescription(),
					date.Format("20060102"),
				),
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.entries.ToCsv(fin.ReportFormatSage, tt.batchDate)
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
		{PolicyID: policy1, IncomeAccount: "44250", RiskCategoryCC: "12345"},
		{PolicyID: policy2, IncomeAccount: "40200", RiskCategoryCC: "67890"},
		{PolicyID: policy3, IncomeAccount: "40200", RiskCategoryCC: "67890"},
	}
	blocks := entries.MakeBlocks()
	ms.Equal(2, len(blocks))

	ms.Equal(1, len(blocks["4425012345"]))
	ms.Equal(policy1, blocks["4425012345"][0].PolicyID)

	ms.Equal(2, len(blocks["4020067890"]))
	ms.Equal(policy2, blocks["4020067890"][0].PolicyID)
	ms.Equal(policy3, blocks["4020067890"][1].PolicyID)

	policy4 := domain.GetUUID()
	policy5 := domain.GetUUID()
	entries = LedgerEntries{
		{PolicyID: policy4, IncomeAccount: "40200", RiskCategoryCC: "67890", Type: LedgerEntryTypeClaim, PolicyType: api.PolicyTypeHousehold},
		{PolicyID: policy5, IncomeAccount: "40200", RiskCategoryCC: "67890", Type: LedgerEntryTypeClaim, PolicyType: api.PolicyTypeTeam},
	}
	blocks = entries.MakeBlocks()
	ms.Equal(2, len(blocks))

	keys := make([]string, 0, len(blocks))
	for k := range blocks {
		keys = append(keys, k)
	}

	ms.ElementsMatch(
		[]string{
			string(api.PolicyTypeHousehold) + "4020067890",
			string(api.PolicyTypeTeam) + "4020067890",
		},
		keys,
	)

	ms.Equal(policy4, blocks[string(api.PolicyTypeHousehold)+"4020067890"][0].PolicyID)
	ms.Equal(policy5, blocks[string(api.PolicyTypeTeam)+"4020067890"][0].PolicyID)
}

func (ms *ModelSuite) TestLedgerEntry_balanceDescription() {
	parentEntity := CreateEntityFixture(ms.DB)
	subEntity := CreateEntityFixture(ms.DB)
	subEntity.ParentEntity = parentEntity.Code
	ms.NoError(ms.DB.Update(&subEntity))

	tests := []struct {
		name  string
		entry LedgerEntry
		want  string
	}{
		{
			name: "no parent entity",
			entry: LedgerEntry{
				EntityCode:       parentEntity.Code,
				RiskCategoryName: "cat1",
				Type:             LedgerEntryTypeNewCoverage,
			},
			want: fmt.Sprintf("Total %s cat1 Premiums", parentEntity.Code),
		},
		{
			name: "has parent entity",
			entry: LedgerEntry{
				EntityCode:       subEntity.Code,
				RiskCategoryName: "cat2",
				Type:             LedgerEntryTypeNewCoverage,
			},
			want: fmt.Sprintf("Total %s cat2 Premiums", parentEntity.Code),
		},
		{
			name: "household claims",
			entry: LedgerEntry{
				EntityCode:       subEntity.Code,
				RiskCategoryName: "cat2",
				PolicyType:       api.PolicyTypeHousehold,
				Type:             LedgerEntryTypeClaimAdjustment,
			},
			want: fmt.Sprintf("Total %s cat2 Claims", api.PolicyTypeHousehold),
		},
		{
			name: "team claims",
			entry: LedgerEntry{
				EntityCode:       subEntity.Code,
				RiskCategoryName: "cat2",
				PolicyType:       api.PolicyTypeTeam,
				Type:             LedgerEntryTypeClaimAdjustment,
			},
			want: fmt.Sprintf("Total %s cat2 Claims", api.PolicyTypeTeam),
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			ms.Equal(tt.want, tt.entry.balanceDescription())
		})
	}
}

func (ms *ModelSuite) Test_NewLedgerEntry() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 2, ClaimsPerPolicy: 1})
	householdPolicy := f.Policies[0]
	householdPolicyItem := householdPolicy.Items[0]
	ms.NoError(householdPolicyItem.SetAccountablePerson(ms.DB, f.Users[0].ID))
	ms.NoError(ms.DB.Update(&householdPolicyItem), "error updating householdPolicyItem for test")

	householdPolicyClaim := f.Policies[0].Claims[0]
	ms.False(uuid.Nil == householdPolicyClaim.ID, "householdPolicyClaim is not hydrated")
	householdPolicyClaim.LoadClaimItems(ms.DB, true)

	teamPolicy := ConvertPolicyType(ms.DB, f.Policies[1])
	teamPolicyItem := teamPolicy.Items[0]
	ms.NoError(teamPolicyItem.SetAccountablePerson(ms.DB, f.Users[1].ID))

	tests := []struct {
		name      string
		policy    Policy
		item      *Item
		claim     *Claim
		claimItem *ClaimItem
	}{
		{
			name:      "household policy item with claim",
			policy:    householdPolicy,
			item:      &householdPolicyItem,
			claim:     &householdPolicyClaim,
			claimItem: &householdPolicyClaim.ClaimItems[0],
		},
		{
			name:   "team policy item no claim",
			policy: teamPolicy,
			item:   &teamPolicyItem,
		},
		{
			name:   "policy only",
			policy: teamPolicy,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			accPersonName := "John Doe"
			le := NewLedgerEntry(accPersonName, tt.policy, tt.item, tt.claim)

			ms.Equal(accPersonName, le.Name, "Name is incorrect")

			ms.Equal(tt.policy.ID, le.PolicyID, "PolicyID is incorrect")
			ms.WithinDuration(time.Now().UTC(), le.DateSubmitted, time.Minute, "DateSubmitted is incorrect")
			ms.Equal(tt.policy.Type, le.PolicyType, "PolicyType is incorrect")
			if tt.policy.Type == api.PolicyTypeTeam {
				ms.Equal(tt.policy.Account, le.AccountNumber, "AccountNumber is incorrect")
				ms.Equal(tt.policy.CostCenter+accountSeparator+tt.policy.AccountDetail, le.CostCenter,
					"CostCenter is incorrect")
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
				ms.Equal(tt.policy.Name, le.PolicyName, "PolicyName is incorrect")
			}

			if tt.claim == nil {
				ms.False(le.ClaimID.Valid, "ClaimID is not nil")
			} else {
				ms.Equal(tt.claim.ID, le.ClaimID.UUID, "ClaimID is incorrect")
				ms.Equal(string(tt.claimItem.PayoutOption), le.ClaimPayoutOption, "ClaimPayoutOption is incorrect")
			}
		})
	}
}

func (ms *ModelSuite) TestLedgerEntry_getDescription() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 2, ClaimsPerPolicy: 1, UsersPerPolicy: 2})
	hhPolicy := f.Policies[0]
	hhPolicyItem := hhPolicy.Items[0]
	hhPolicy.LoadMembers(ms.DB, false)

	// Give the household item an accountable person
	hhAccPerson := hhPolicy.Members[1]
	ms.NoError(hhPolicyItem.SetAccountablePerson(ms.DB, hhAccPerson.ID))
	ms.NoError(ms.DB.Update(&hhPolicyItem), "error updating household policy item for test")

	hhPolicyClaim := f.Policies[0].Claims[0]
	ms.False(uuid.Nil == hhPolicyClaim.ID, "householdPolicyClaim is not hydrated")

	teamPolicy := ConvertPolicyType(ms.DB, f.Policies[1])
	teamPolicyItem := teamPolicy.Items[0]
	teamPolicy.LoadMembers(ms.DB, false)

	// Give the team item an accountable person
	teamAccPerson := teamPolicy.Members[1]
	ms.NoError(teamPolicyItem.SetAccountablePerson(ms.DB, teamPolicy.Members[1].ID))
	ms.NoError(ms.DB.Update(&teamPolicyItem), "error updating team policy item for test")

	// Create new Ledger Entries for each policy
	hhAccPersName := hhAccPerson.GetName().String()
	hhEntry := NewLedgerEntry(hhAccPersName, hhPolicy, &hhPolicyItem, nil)
	hhEntry.Type = LedgerEntryTypeNewCoverage

	teamAccPersName := teamAccPerson.GetName().String()
	teamEntry := NewLedgerEntry(teamAccPersName, teamPolicy, &teamPolicyItem, nil)
	teamEntry.Type = LedgerEntryTypeCoverageRefund

	tests := []struct {
		name  string
		entry LedgerEntry
		item  Item
		want  string
	}{
		{
			name:  "household policy item",
			entry: hhEntry,
			item:  hhPolicyItem,
			want:  fmt.Sprintf("%s / %s", `Coverage premium: Add`, hhPolicy.Name),
		},
		{
			name:  "team policy item",
			entry: teamEntry,
			item:  teamPolicyItem,
			want:  fmt.Sprintf("%s / %s (%s)", `Coverage reimbursement: Remove`, teamPolicy.Name, teamAccPersName),
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.entry.getDescription()

			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestLedgerEntry_getItemName() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 2, ClaimsPerPolicy: 1, UsersPerPolicy: 2})
	hhPolicy := f.Policies[0]
	hhPolicyItem := hhPolicy.Items[0]
	hhPolicy.LoadMembers(ms.DB, false)

	// Give the household item an accountable person
	hhAccPerson := hhPolicy.Members[1]
	ms.NoError(hhPolicyItem.SetAccountablePerson(ms.DB, hhAccPerson.ID))
	ms.NoError(ms.DB.Update(&hhPolicyItem), "error updating household policy item for test")

	hhPolicyClaim := f.Policies[0].Claims[0]
	ms.False(uuid.Nil == hhPolicyClaim.ID, "householdPolicyClaim is not hydrated")

	teamPolicy := ConvertPolicyType(ms.DB, f.Policies[1])
	teamPolicyItem := teamPolicy.Items[0]
	teamPolicy.LoadMembers(ms.DB, false)

	// Give the team item an accountable person
	teamAccPerson := teamPolicy.Members[1]
	ms.NoError(teamPolicyItem.SetAccountablePerson(ms.DB, teamPolicy.Members[1].ID))
	ms.NoError(ms.DB.Update(&teamPolicyItem), "error updating team policy item for test")

	// Create new Ledger Entries for each policy
	hhAccPersName := hhAccPerson.GetName().String()
	hhEntry := NewLedgerEntry(hhAccPersName, hhPolicy, &hhPolicyItem, nil)
	hhEntry.Type = LedgerEntryTypeNewCoverage

	teamAccPersName := teamAccPerson.GetName().String()
	teamEntry := NewLedgerEntry(teamAccPersName, teamPolicy, &teamPolicyItem, nil)
	teamEntry.Type = LedgerEntryTypeCoverageRefund

	tests := []struct {
		name  string
		entry LedgerEntry
		want  string
	}{
		{
			name:  "household policy item",
			entry: hhEntry,
			want:  hhPolicyItem.Name,
		},
		{
			name:  "team policy item",
			entry: teamEntry,
			want:  teamPolicyItem.Name,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.entry.getItemName(ms.DB)

			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestLedgerEntry_getReference() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 2, ClaimsPerPolicy: 1, UsersPerPolicy: 2})
	hhPolicy := f.Policies[0]
	hhPolicyItem := hhPolicy.Items[0]
	hhPolicy.LoadMembers(ms.DB, false)

	// Give the household item an accountable person
	hhAccPerson := hhPolicy.Members[1]
	hhAccPersonName := hhAccPerson.GetName().String()
	ms.NoError(hhPolicyItem.SetAccountablePerson(ms.DB, hhAccPerson.ID))
	ms.NoError(ms.DB.Update(&hhPolicyItem), "error updating household policy item for test")

	hhPolicyClaim := f.Policies[0].Claims[0]
	ms.False(uuid.Nil == hhPolicyClaim.ID, "householdPolicyClaim is not hydrated")

	teamPolicy := ConvertPolicyType(ms.DB, f.Policies[1])
	teamPolicyItem := teamPolicy.Items[0]
	teamPolicy.LoadMembers(ms.DB, false)

	// Give the team item an accountable person
	ms.NoError(teamPolicyItem.SetAccountablePerson(ms.DB, teamPolicy.Members[1].ID))
	ms.NoError(ms.DB.Update(&teamPolicyItem), "error updating team policy item for test")

	// Create new Ledger Entries for each policy
	hhEntry := NewLedgerEntry(hhAccPersonName, hhPolicy, &hhPolicyItem, nil)
	hhEntry.Type = LedgerEntryTypeNewCoverage

	teamEntry := NewLedgerEntry("", teamPolicy, &teamPolicyItem, nil)
	teamEntry.Type = LedgerEntryTypeCoverageRenewal
	teamEntry.AccountNumber = "TAcc"
	teamEntry.CostCenter = "TCC"

	tests := []struct {
		name  string
		entry LedgerEntry
		item  Item
		want  string
	}{
		{
			name:  "household policy item",
			entry: hhEntry,
			item:  hhPolicyItem,
			want:  fmt.Sprintf("MC %s / %s", hhEntry.HouseholdID, hhAccPersonName),
		},
		{
			name:  "team policy item",
			entry: teamEntry,
			item:  teamPolicyItem,
			want: fmt.Sprintf("%s %s%s / %s",
				teamEntry.EntityCode, teamEntry.AccountNumber, teamEntry.CostCenter, teamPolicy.Name),
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := getReference(tt.entry)

			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestLedgerEntries_Reconcile() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2, ClaimsPerPolicy: 2, ClaimItemsPerClaim: 1})
	ctx := CreateTestContext(CreateAdminUsers(ms.DB)[AppRoleSteward])

	march := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	april := time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC)

	datesSubmitted := []time.Time{march, april}
	datesEntered := []nulls.Time{nulls.NewTime(april), {}}

	itemEntries := make(LedgerEntries, len(f.Items))
	for i := range f.Items {
		ms.NoError(f.Items[i].Approve(ctx, false))

		ms.NoError(ms.DB.Where("item_id = ?", f.Items[i].ID).First(&itemEntries[i]))
		itemEntries[i].DateSubmitted = datesSubmitted[i]
		itemEntries[i].DateEntered = datesEntered[i]
		ms.NoError(ms.DB.Update(&itemEntries[i]))
	}

	claimEntries := make(LedgerEntries, len(f.Claims))
	for i, claim := range f.Claims {
		claim = UpdateClaimStatus(ms.DB, claim, api.ClaimStatusReview3, "")
		ms.NoError(claim.Approve(ctx))

		ms.NoError(ms.DB.Where("claim_id = ?", claim.ID).First(&claimEntries[i]))
		claimEntries[i].DateSubmitted = datesSubmitted[i]
		claimEntries[i].DateEntered = datesEntered[i]
		ms.NoError(ms.DB.Update(&claimEntries[i]))
	}

	empty := LedgerEntries{}

	tests := []struct {
		name    string
		entries LedgerEntries
	}{
		{
			name:    "empty list",
			entries: empty,
		},
		{
			name:    "item ledger entries",
			entries: itemEntries,
		},
		{
			name:    "claim ledger entries",
			entries: claimEntries,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			err := tt.entries.Reconcile(ctx)
			ms.NoError(err)

			for _, e := range tt.entries {
				var after LedgerEntry
				ms.NoError(ms.DB.Find(&after, e.ID))
				ms.True(after.DateEntered.Valid, "DateEntered was not set")

				if after.ClaimID.Valid {
					after.LoadClaim(ms.DB)
					ms.Equal(api.ClaimStatusPaid, after.Claim.Status, "claim Status was not set to Paid")
				}
			}
		})
	}
}

func (ms *ModelSuite) Test_AdjustLedgerAmount() {
	tests := []struct {
		name       string
		entryType  LedgerEntryType
		amount     api.Currency
		wantAmount api.Currency
	}{
		{
			name:       "new coverage, over $1",
			entryType:  LedgerEntryTypeNewCoverage,
			amount:     500,
			wantAmount: 500,
		},
		{
			name:       "new coverage, under $1",
			entryType:  LedgerEntryTypeNewCoverage,
			amount:     50,
			wantAmount: 50,
		},
		{
			name:       "refund, over $1",
			entryType:  LedgerEntryTypeCoverageRefund,
			amount:     -500,
			wantAmount: -500,
		},
		{
			name:       "refund, under $1",
			entryType:  LedgerEntryTypeCoverageRefund,
			amount:     -50,
			wantAmount: -0,
		},
		{
			name:       "coverage change, positive, over $1",
			entryType:  LedgerEntryTypeCoverageChange,
			amount:     600,
			wantAmount: 600,
		},
		{
			name:       "coverage change, positive, under $1",
			entryType:  LedgerEntryTypeCoverageChange,
			amount:     50,
			wantAmount: 50,
		},
		{
			name:       "coverage change, negative, over $1",
			entryType:  LedgerEntryTypeCoverageChange,
			amount:     -700,
			wantAmount: -700,
		},
		{
			name:       "coverage change, negative, under $1",
			entryType:  LedgerEntryTypeCoverageChange,
			amount:     -30,
			wantAmount: -0,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got, err := adjustLedgerAmount(tt.amount, tt.entryType)
			ms.NoError(err)

			ms.Equal(tt.wantAmount, got, "adjustment is incorrect")
		})
	}
}

func getReference(le LedgerEntry) string {
	// For household policies
	if le.PolicyType == api.PolicyTypeHousehold {
		ref := fmt.Sprintf("MC %s", le.HouseholdID)

		if le.Name == "" {
			return ref
		}

		return fmt.Sprintf("%s / %s", ref, le.Name)
	}

	// For non-household policies
	if le.PolicyName == "" {
		return fmt.Sprintf("%s %s%s", le.EntityCode, le.AccountNumber, le.CostCenter)
	}

	return fmt.Sprintf("%s %s%s / %s",
		le.EntityCode, le.AccountNumber, le.CostCenter, le.PolicyName)
}
