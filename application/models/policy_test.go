package models

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestPolicy_Validate() {
	t := ms.T()
	tests := []struct {
		name     string
		Policy   Policy
		wantErr  bool
		errField string
	}{
		{
			name: "invalid",
			Policy: Policy{
				Name: "my policy",
				Type: "invalid",
			},
			wantErr:  true,
			errField: "Policy.Type",
		},
		{
			name: "missing type",
			Policy: Policy{
				Name: "my policy",
			},
			wantErr:  true,
			errField: "Policy.Type",
		},
		{
			name: "household type, should not have cost center",
			Policy: Policy{
				Name:        "my policy",
				Type:        api.PolicyTypeHousehold,
				HouseholdID: nulls.NewString("abc123"),
				CostCenter:  "forbidden",
			},
			wantErr:  true,
			errField: "Policy.CostCenter",
		},
		{
			name: "household type, should not have account",
			Policy: Policy{
				Name:        "my policy",
				Type:        api.PolicyTypeHousehold,
				HouseholdID: nulls.NewString("abc123"),
				Account:     "forbidden",
			},
			wantErr:  true,
			errField: "Policy.Account",
		},
		{
			name: "team type, should not have household id",
			Policy: Policy{
				Name:         "my policy",
				Type:         api.PolicyTypeTeam,
				HouseholdID:  nulls.NewString("abc123"),
				CostCenter:   "abc123",
				Account:      "123456",
				EntityCodeID: domain.GetUUID(),
			},
			wantErr:  true,
			errField: "Policy.HouseholdID",
		},
		{
			name: "team type, should have either account or cost center",
			Policy: Policy{
				Name:         "my policy",
				Type:         api.PolicyTypeTeam,
				EntityCodeID: domain.GetUUID(),
			},
			wantErr:  true,
			errField: "Policy.CostCenter",
		},
		{
			name: "incorrect entity code id",
			Policy: Policy{
				Name:         "my policy",
				Type:         api.PolicyTypeHousehold,
				HouseholdID:  nulls.NewString("abc123"),
				EntityCodeID: domain.GetUUID(),
				CostCenter:   "abc123",
				Account:      "123456",
			},
			wantErr:  true,
			errField: "Policy.EntityCodeID",
		},
		{
			name: "valid household type",
			Policy: Policy{
				Name:         "my policy",
				Type:         api.PolicyTypeHousehold,
				HouseholdID:  nulls.NewString("123456"),
				EntityCodeID: HouseholdEntityID(),
			},
			wantErr:  false,
			errField: "",
		},
		{
			name: "valid team type",
			Policy: Policy{
				Name:         "my policy",
				Type:         api.PolicyTypeTeam,
				CostCenter:   "abc123",
				Account:      "123456",
				EntityCodeID: domain.GetUUID(),
			},
			wantErr:  false,
			errField: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.Policy.Validate(DB)
			if tt.wantErr {
				if vErr.Count() == 0 {
					t.Errorf("Expected an error, but did not get one")
				} else if len(vErr.Get(tt.errField)) == 0 {
					t.Errorf("Expected an error on field %v, but got none (errors: %+v)", tt.errField, vErr.Errors)
				}
			} else if vErr.HasAny() {
				t.Errorf("Unexpected error: %+v", vErr)
			}
		})
	}
}

func (ms *ModelSuite) TestPolicy_CreateTeam() {
	t := ms.T()

	pf := CreatePolicyFixtures(ms.DB, FixturesConfig{NumberOfEntityCodes: 1})
	entCode := pf.EntityCodes[0]

	uf := CreateUserFixtures(ms.DB, 2)
	user := uf.Users[0]

	goodPolicy := Policy{
		Name:         "my policy",
		CostCenter:   randStr(8),
		Account:      randStr(8),
		EntityCodeID: entCode.ID,
	}

	missingCC := goodPolicy
	missingCC.CostCenter = ""
	missingCC.Account = ""

	missingEntCode := goodPolicy
	missingEntCode.EntityCodeID = uuid.Nil

	tests := []struct {
		name    string
		user    User
		policy  Policy
		wantErr bool
	}{
		{
			name:    "empty user",
			user:    User{},
			policy:  goodPolicy,
			wantErr: true,
		},
		{
			name:    "missing CostCenter and Account",
			user:    user,
			policy:  missingCC,
			wantErr: true,
		},
		{
			name:    "missing EntityCode",
			user:    user,
			policy:  missingEntCode,
			wantErr: true,
		},
		{
			name:    "good policy to be created",
			user:    user,
			policy:  goodPolicy,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := CreateTestContext(tt.user)
			err := tt.policy.CreateTeam(context)
			if tt.wantErr {
				ms.Error(err)
				return
			}

			ms.NoError(err)

			dbPolicy := Policy{}
			err = ms.DB.Where("id = ?", &tt.policy.ID).First(&dbPolicy)

			ms.NoError(err, "error trying to find resulting policy")
			ms.Equal(tt.policy.Account, dbPolicy.Account)
			ms.Equal(tt.user.EmailOfChoice(), dbPolicy.Email)
			ms.Equal(api.PolicyTypeTeam, dbPolicy.Type)

			policyUsers := PolicyUsers{}
			err = ms.DB.Where("user_id = ?", tt.user.ID).All(&policyUsers)
			ms.NoError(err, "error trying to find resulting policyUsers")
			ms.Len(policyUsers, 1, "incorrect number of policyUsers")
		})
	}
}

func (ms *ModelSuite) TestPolicy_LoadMembers() {
	f := CreatePolicyFixtures(ms.DB, FixturesConfig{})
	policy := f.Policies[0]

	policy.LoadMembers(ms.DB, false)
	ms.Len(policy.Members, 1)
}

func idsInOrder(id1, id2 uuid.UUID) string {
	const template = `%s||%s`
	id1S := id1.String()
	id2S := id2.String()
	if id1S <= id2S {
		return fmt.Sprintf(template, id1S, id2S)
	}

	return fmt.Sprintf(template, id2S, id1S)
}

func (ms *ModelSuite) TestPolicy_GetPolicyUserIDs() {
	f := CreatePolicyFixtures(ms.DB, FixturesConfig{UsersPerPolicy: 2})
	policy := f.Policies[0]
	got := policy.GetPolicyUserIDs(ms.DB, true)
	ms.Len(got, 2, "incorrect number of PolicyUserIDs")

	var polUsers PolicyUsers
	ms.NoError(ms.DB.Where("policy_id = ?", policy.ID).All(&polUsers),
		"error fetching PolicyUsers to verify")

	wantIDs := idsInOrder(polUsers[0].ID, polUsers[1].ID)
	gotIDs := idsInOrder(got[0], got[1])
	ms.Equal(wantIDs, gotIDs, "incorrect PolicyUser ID(s)")
}

func (ms *ModelSuite) TestPolicy_LoadDependents() {
	f := CreatePolicyFixtures(ms.DB, FixturesConfig{DependentsPerPolicy: 1})
	policy := f.Policies[0]

	policy.LoadDependents(ms.DB, false)
	ms.Len(policy.Dependents, 1)
}

func (ms *ModelSuite) TestPolicy_LoadInvites() {
	f := CreatePolicyFixtures(ms.DB, FixturesConfig{InvitesPerPolicy: 2})
	policy := f.Policies[0]

	policy.LoadInvites(ms.DB, false)
	ms.Len(policy.Invites, 2)
}

func (ms *ModelSuite) TestPolicy_itemCoverageTotals() {
	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 2,
		ItemsPerPolicy:      5,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	policy.LoadItems(ms.DB, false)
	items := policy.Items

	// give two items a dependant and calculate expected values
	dependant := policy.Dependents[0]
	coverageForPolicy := 0
	coverageForDep := 0
	for i := range items {
		// Set to approved
		if i < 4 {
			items[i] = UpdateItemStatus(ms.DB, items[i], api.ItemCoverageStatusApproved, "")
			coverageForPolicy += items[i].CoverageAmount
		}

		if i == 2 || i == 3 {
			items[i].PolicyDependentID = nulls.NewUUID(dependant.ID)
			ms.NoError(ms.DB.Update(&items[i]), "error trying to change item DependantID")
			coverageForDep += items[i].CoverageAmount
		}
	}

	policy.Items = Items{} // ensure the LoadItems gets called

	got := policy.itemCoverageTotals(ms.DB)

	ms.Equal(coverageForPolicy, got[policy.ID], "incorrect policy coverage total")
	ms.Equal(coverageForDep, got[dependant.ID], "incorrect dependant coverage total")
	ms.Greater(coverageForPolicy, coverageForDep, "double checking exposed a problem with the test design")

	// Note this includes the dependant total twice, which is OK for testing purposes
	gotTotal := 0
	for _, v := range got {
		gotTotal += v
	}

	want := coverageForPolicy + coverageForDep
	ms.Equal(want, gotTotal, "incorrect coverage grand total")
}

func (ms *ModelSuite) TestPolicy_Compare() {
	e := CreateEntityFixture(ms.DB)

	oldPolicy := Policy{
		Type:         api.PolicyTypeTeam,
		HouseholdID:  nulls.NewString("abc123"),
		CostCenter:   "xyz789",
		Account:      "123457890",
		EntityCodeID: e.ID,
		Notes:        randStr(19),
	}

	f := CreatePolicyFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 1})
	newPolicy := f.Policies[0]

	tests := []struct {
		name string
		new  Policy
		old  Policy
		want []FieldUpdate
	}{
		{
			name: "1",
			new:  f.Policies[0],
			old:  oldPolicy,
			want: []FieldUpdate{
				{
					FieldName: "Name",
					OldValue:  oldPolicy.Name,
					NewValue:  newPolicy.Name,
				},
				{
					FieldName: "Type",
					OldValue:  string(oldPolicy.Type),
					NewValue:  string(newPolicy.Type),
				},
				{
					FieldName: "HouseholdID",
					OldValue:  oldPolicy.HouseholdID.String,
					NewValue:  newPolicy.HouseholdID.String,
				},
				{
					FieldName: "CostCenter",
					OldValue:  oldPolicy.CostCenter,
					NewValue:  newPolicy.CostCenter,
				},
				{
					FieldName: "Account",
					OldValue:  oldPolicy.Account,
					NewValue:  newPolicy.Account,
				},
				{
					FieldName: "EntityCodeID",
					OldValue:  oldPolicy.EntityCodeID.String(),
					NewValue:  newPolicy.EntityCodeID.String(),
				},
				{
					FieldName: "Notes",
					OldValue:  oldPolicy.Notes,
					NewValue:  newPolicy.Notes,
				},
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.new.Compare(tt.old)
			ms.ElementsMatch(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestPolicy_MemberHasEmail() {
	db := ms.DB

	f := CreatePolicyFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 1})
	policy := f.Policies[0]
	member := policy.Members[0]

	tests := []struct {
		name   string
		policy Policy
		email  string
		want   bool
	}{
		{
			name:   "no match",
			policy: policy,
			email:  "unique1@example.org",
			want:   false,
		},
		{
			name:   "has match",
			policy: policy,
			email:  member.Email,
			want:   true,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.policy.MemberHasEmail(db, tt.email)
			ms.Equal(tt.want, got, "incorrect return value")
		})
	}
}

func (ms *ModelSuite) TestPolicy_NewHistory() {
	f := CreatePolicyFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 1})
	policy := f.Policies[0]
	user := f.Users[0]

	const newHouseholdID = "NEW01234"
	const newEntityCodeID = "3eb5d328-0831-4d3f-a260-db0531f29730"

	tests := []struct {
		name   string
		policy Policy
		user   User
		update FieldUpdate
		want   PolicyHistory
	}{
		{
			name:   "HouseholdID",
			policy: policy,
			user:   user,
			update: FieldUpdate{
				FieldName: "HouseholdID",
				OldValue:  policy.HouseholdID.String,
				NewValue:  newHouseholdID,
			},
			want: PolicyHistory{
				PolicyID:  policy.ID,
				UserID:    user.ID,
				Action:    api.HistoryActionUpdate,
				FieldName: "HouseholdID",
				OldValue:  policy.HouseholdID.String,
				NewValue:  newHouseholdID,
			},
		},
		{
			name:   "EntityCodeID",
			policy: policy,
			user:   user,
			update: FieldUpdate{
				FieldName: "EntityCodeID",
				OldValue:  policy.EntityCodeID.String(),
				NewValue:  newEntityCodeID,
			},
			want: PolicyHistory{
				PolicyID:  policy.ID,
				UserID:    user.ID,
				Action:    api.HistoryActionUpdate,
				FieldName: "EntityCodeID",
				OldValue:  policy.EntityCodeID.String(),
				NewValue:  newEntityCodeID,
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.policy.NewHistory(CreateTestContext(tt.user), api.HistoryActionUpdate, tt.update)
			ms.False(tt.want.NewValue == tt.want.OldValue, "test isn't correctly checking a field update")
			ms.Equal(tt.want.PolicyID, got.PolicyID, "PolicyID is not correct")
			ms.Equal(tt.want.UserID, got.UserID, "UserID is not correct")
			ms.Equal(tt.want.Action, got.Action, "Action is not correct")
			ms.Equal(tt.want.FieldName, got.FieldName, "FieldName is not correct")
			ms.Equal(tt.want.OldValue, got.OldValue, "OldValue is not correct")
			ms.Equal(tt.want.NewValue, got.NewValue, "NewValue is not correct")
		})
	}
}

func (ms *ModelSuite) TestPolicy_calculateAnnualPremium() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 2})

	secondItem := createItemFixture(ms.DB, f.Policies[1].ID, CreateCategoryFixtures(ms.DB, 1).ItemCategories[0].ID)
	secondItem.CoverageAmount = int(float64(domain.Env.PremiumMinimum) / domain.Env.PremiumFactor)
	ms.NoError(secondItem.Update(CreateTestContext(f.Users[0])))
	f.Policies[1].LoadItems(ms.DB, true)

	// Use a fresh copy, since the UUT does not expect pre-hydration
	firstPolicy := Policy{ID: f.Policies[0].ID}
	ms.NoError(ms.DB.Reload(&firstPolicy))

	secondPolicy := Policy{ID: f.Policies[1].ID}
	ms.NoError(ms.DB.Reload(&secondPolicy))

	tests := []struct {
		name   string
		policy Policy
		want   api.Currency
	}{
		{
			name:   "one item, below minimum",
			policy: firstPolicy,
			want:   api.Currency(domain.Env.PremiumMinimum),
		},
		{
			name:   "two items, above minimum",
			policy: secondPolicy,
			want:   f.Policies[1].Items[0].CalculateAnnualPremium() + f.Policies[1].Items[1].CalculateAnnualPremium(),
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.policy.calculateAnnualPremium(ms.DB)
			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestPolicy_ConvertToAPI() {
	fConfig := FixturesConfig{
		DependentsPerPolicy: 1,
		ClaimsPerPolicy:     1,
		InvitesPerPolicy:    2,
	}
	f := CreateItemFixtures(ms.DB, fConfig)

	policy := f.Policies[0]
	policy = ConvertPolicyType(ms.DB, policy)

	got := policy.ConvertToAPI(ms.DB, false)

	ms.Equal(policy.ID, got.ID, "ID is not correct")
	ms.Equal(policy.Name, got.Name, "Name is not correct")
	ms.Equal(policy.Type, got.Type, "Type is not correct")
	ms.Equal(policy.HouseholdID.String, got.HouseholdID, "HouseholdID is not correct")
	ms.Equal(policy.CostCenter, got.CostCenter, "CostCenter is not correct")
	ms.Equal(policy.Account, got.Account, "Account is not correct")
	ms.Equal(policy.AccountDetail, got.AccountDetail, "AccountDetail is not correct")
	ms.Equal(policy.EntityCode.ConvertToAPI(ms.DB, false), got.EntityCode, "EntityCode is not correct")
	ms.Equal(policy.CreatedAt, got.CreatedAt, "CreatedAt is not correct")
	ms.Equal(policy.UpdatedAt, got.UpdatedAt, "UpdatedAt is not correct")
	ms.Equal(0, len(got.Dependents), "Dependents should not be hydrated")
	ms.Equal(0, len(got.Claims), "Claims should not be hydrated")
	ms.Equal(0, len(got.Invites), "Invites should not be hydrated")

	ms.Greater(len(got.Members), 0, "test should be revised, fixture has no Members")
	ms.Len(got.Members, len(got.Members), "Members is not correct length")

	got = policy.ConvertToAPI(ms.DB, true)

	ms.Greater(len(f.PolicyDependents), 0, "test should be revised, fixture has no Dependents")
	ms.Len(got.Dependents, len(f.PolicyDependents), "Files is not correct length")

	ms.Greater(len(f.Claims), 0, "test should be revised, fixture has no Claims")
	ms.Len(got.Claims, len(f.Claims), "Claims is not correct length")

	ms.Greater(len(f.PolicyUserInvites), 0, "test should be revised, fixture has no Invites")
	ms.Len(got.Invites, len(f.PolicyUserInvites), "Invites is not correct length")
}

func (ms *ModelSuite) TestPolicies_Query() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 4, ItemsPerPolicy: 2, UsersPerPolicy: 2})

	corpPolicy := ConvertPolicyType(ms.DB, f.Policies[0])

	f.Policies[0].Members[0].FirstName = "Matthew"
	f.Policies[0].Members[0].LastName = "Smythe"
	ms.NoError(ms.DB.Update(&f.Policies[0].Members[0]))

	f.Policies[1].Members[0].FirstName = "Hew"
	f.Policies[1].Members[0].LastName = "Smith"
	ms.NoError(ms.DB.Update(&f.Policies[1].Members[0]))

	f.Policies[1].Members[1].LastName = "Smith"
	ms.NoError(ms.DB.Update(&f.Policies[1].Members[1]))

	f.Policies[2].Members[0].FirstName = "John"
	ms.NoError(ms.DB.Update(&f.Policies[2].Members[0]))

	f.Policies[3].Members[0].FirstName = "John"
	ms.NoError(ms.DB.Update(&f.Policies[3].Members[0]))

	tests := []struct {
		name                 string
		query                string
		wantNumberOfPolicies int
	}{
		{
			name:                 "none found",
			query:                "search=not gonna find this one",
			wantNumberOfPolicies: 0,
		},
		{
			name:                 "first name",
			query:                "search=matthew",
			wantNumberOfPolicies: 1,
		},
		{
			name:                 "last name",
			query:                "search=smith",
			wantNumberOfPolicies: 1,
		},
		{
			name:                 "partial first name",
			query:                "search=matt",
			wantNumberOfPolicies: 1,
		},
		{
			name:                 "full name",
			query:                "search=matthew smythe",
			wantNumberOfPolicies: 1,
		},
		{
			name:                 "partial on full name",
			query:                "search=hew sm",
			wantNumberOfPolicies: 2,
		},
		{
			name:                 "policy name",
			query:                "search=" + corpPolicy.Name,
			wantNumberOfPolicies: 1,
		},
		{
			name:                 "cost center",
			query:                "search=" + corpPolicy.CostCenter,
			wantNumberOfPolicies: 1,
		},
		{
			name:                 "household ID",
			query:                "search=" + f.Policies[1].HouseholdID.String,
			wantNumberOfPolicies: 1,
		},
		{
			name:                 "limit 2",
			query:                "search=john&limit=2",
			wantNumberOfPolicies: 2,
		},
		{
			name:                 "limit 1",
			query:                "search=john&limit=1",
			wantNumberOfPolicies: 1,
		},
		{
			name:                 "only active",
			query:                "filter=active:true",
			wantNumberOfPolicies: 0,
		},
		{
			name:                 "only inactive",
			query:                "filter=active:false",
			wantNumberOfPolicies: 4,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			var policies Policies

			values, _ := url.ParseQuery(tt.query)
			query := api.NewQueryParams(buffalo.ParamValues(values))

			p, err := policies.Query(ms.DB, query)
			ms.NoError(err)
			ms.Equal(tt.wantNumberOfPolicies, len(policies), "got wrong number of policies")
			ms.NotNil(p, "should be a value")
			ms.Equal(p.Page, 1, "should default to page 1")
		})
	}
}

func (ms *ModelSuite) TestPolicy_ProcessAnnualCoverage() {
	year := time.Now().UTC().Year()

	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 5})

	f.Items[0].PaidThroughYear = year
	f.Items[2].RiskCategoryID = RiskCategoryMobileID()
	for i := range f.Items {
		UpdateItemStatus(ms.DB, f.Items[i], api.ItemCoverageStatusApproved, "")
	}

	err := f.Policies[0].ProcessAnnualCoverage(ms.DB, year)
	ms.NoError(err)

	var l LedgerEntries
	ms.NoError(l.FindRenewals(ms.DB, year))

	// should be one for each risk category
	ms.Equal(2, len(l))

	// do it again to make sure it doesn't make double ledger entries
	err = f.Policies[0].ProcessAnnualCoverage(ms.DB, year)
	ms.NoError(err)

	var l2 LedgerEntries
	ms.NoError(l2.FindRenewals(ms.DB, year))
	ms.Equal(2, len(l2))
}

func (ms *ModelSuite) TestPolicy_currentCoverage() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 5})
	policy := f.Policies[0]
	totalCoverage := 0
	for i := range f.Items {
		UpdateItemStatus(ms.DB, f.Items[i], api.ItemCoverageStatusApproved, "")
		totalCoverage += f.Items[i].CoverageAmount
	}
	ms.Greaterf(totalCoverage, 0, "total coverage did not get calculated properly for test")

	policy.LoadItems(ms.DB, true)

	coverage := policy.currentCoverageTotal(ms.DB)
	ms.Equal(api.Currency(totalCoverage), coverage, "incorrect Coverage for Policy")
}
