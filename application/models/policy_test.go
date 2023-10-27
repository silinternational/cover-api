package models

import (
	"fmt"
	"net/url"
	"strings"
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

func (ms *ModelSuite) TestPolicy_AddDependent() {
	fixtures := CreatePolicyFixtures(ms.DB, FixturesConfig{DependentsPerPolicy: 1})
	policy := fixtures.Policies[0]
	dependent := fixtures.PolicyDependents[0]

	tests := []struct {
		name    string
		policy  Policy
		input   api.PolicyDependentInput
		want    PolicyDependent
		wantErr *api.AppError
	}{
		{
			name:    "incomplete",
			policy:  policy,
			input:   api.PolicyDependentInput{Name: "Simon"},
			wantErr: &api.AppError{Category: api.CategoryUser, Key: api.ErrorValidation},
		},
		{
			name:   "name conflict",
			policy: policy,
			input: api.PolicyDependentInput{
				Name:           dependent.Name,
				Country:        "Narnia",
				Relationship:   dependent.Relationship,
				ChildBirthYear: dependent.ChildBirthYear,
			},
			wantErr: &api.AppError{Category: api.CategoryUser, Key: api.ErrorPolicyDependentDuplicateName},
		},
		{
			name:   "create new",
			policy: policy,
			input: api.PolicyDependentInput{
				Name:         "Simon",
				Country:      "USA",
				Relationship: api.PolicyDependentRelationshipSpouse,
			},
		},
		{
			name:   "reuse existing",
			policy: policy,
			input: api.PolicyDependentInput{
				Name:           dependent.Name,
				Country:        dependent.Country,
				Relationship:   dependent.Relationship,
				ChildBirthYear: dependent.ChildBirthYear,
			},
			want: dependent,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got, err := tt.policy.AddDependent(ms.DB, tt.input)
			if tt.wantErr != nil {
				ms.Error(err)
				AssertSameAppError(ms.T(), *tt.wantErr, err)
				return
			}

			ms.NoError(err)
			ms.Equal(tt.input.Name, got.Name)
			ms.Equal(tt.input.Country, got.Country)
			ms.Equal(tt.input.Relationship, got.Relationship)

			if !tt.want.ID.IsNil() {
				ms.Equal(tt.want.ID, got.ID, "expected ID of existing dependent")
			}
		})
	}
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
			want: f.Policies[1].Items[0].CalculateAnnualPremium(ms.DB) +
				f.Policies[1].Items[1].CalculateAnnualPremium(ms.DB),
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

	// create a policy with no users
	f2 := CreatePolicyFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 1})
	f2.Policies[0].Name = "ABC123"
	Must(ms.DB.Update(&f2.Policies[0]))
	f2.PolicyUsers[0].PolicyID = corpPolicy.ID
	Must(ms.DB.Update(&f2.PolicyUsers[0]))

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
			name:                 "policy with no users",
			query:                "search=" + "ABC",
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
			wantNumberOfPolicies: 5,
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

func (ms *ModelSuite) TestPolicy_ProcessRenewals() {
	now := time.Now().UTC()
	year := now.Year()
	endOfLastMonth := domain.EndOfMonth(now.AddDate(0, -1, 0))

	const annualItems = 4
	annual := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: annualItems})
	annual.Items[2].RiskCategoryID = RiskCategoryMobileID()
	annual.Items[3].RiskCategoryID = RiskCategoryMobileID()
	for i := range annual.Items {
		annual.Items[i].PaidThroughDate = domain.EndOfYear(year - 1)
		annual.Items[i].CoverageAmount = 1000

		UpdateItemStatus(ms.DB, annual.Items[i], api.ItemCoverageStatusApproved, "")
	}

	const monthlyItems = 3
	monthly := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: monthlyItems})
	for i := range monthly.Items {
		monthly.Items[i].RiskCategoryID = RiskCategoryMobileID()
		monthly.Items[i].PaidThroughDate = endOfLastMonth
		monthly.Items[i].CoverageAmount = 12_000
		UpdateItemStatus(ms.DB, monthly.Items[i], api.ItemCoverageStatusApproved, "")
		monthly.ItemCategories[i].BillingPeriod = domain.BillingPeriodMonthly
		monthly.ItemCategories[1].RiskCategoryID = riskCategoryVehicleID
		Must(ms.DB.Update(&monthly.ItemCategories[i]))
	}

	tests := []struct {
		name             string
		policy           Policy
		date             time.Time
		billingPeriod    int
		wantEntriesCount int
		wantPaidThrough  time.Time
		wantAmount       api.Currency
	}{
		{
			name:             "annual",
			policy:           annual.Policies[0],
			date:             now,
			billingPeriod:    domain.BillingPeriodAnnual,
			wantEntriesCount: 2, // two risk categories
			wantPaidThrough:  domain.EndOfYear(year),
			wantAmount:       -20 * annualItems / 2, // divide by the number of risk categories
		},
		{
			name:             "monthly",
			policy:           monthly.Policies[0],
			date:             now,
			billingPeriod:    domain.BillingPeriodMonthly,
			wantEntriesCount: 1, // only one risk category
			wantPaidThrough:  domain.EndOfMonth(now),
			wantAmount:       -20 * monthlyItems,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			ms.NoError(tt.policy.ProcessRenewals(ms.DB, tt.date, tt.billingPeriod))

			var l LedgerEntries
			Must(ms.DB.Where("policy_id = ?", tt.policy.ID).All(&l))
			ms.Equal(tt.wantEntriesCount, len(l))

			var i Items
			Must(ms.DB.Where("policy_id = ?", tt.policy.ID).All(&i))
			for _, ii := range i {
				ms.Equal(tt.wantPaidThrough, ii.PaidThroughDate)
			}

			// do it again to make sure it doesn't make double ledger entries
			ms.NoError(tt.policy.ProcessRenewals(ms.DB, tt.date, tt.billingPeriod))
			var l2 LedgerEntries
			Must(ms.DB.Where("policy_id = ?", tt.policy.ID).All(&l2))
			ms.Equal(tt.wantEntriesCount, len(l2))
			for i := range l2 {
				ms.Equal(tt.wantAmount, l2[i].Amount)
			}
		})
	}
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

func (ms *ModelSuite) TestPolicy_CreateRenewalLedgerEntry() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 1})
	f.Items[0].RiskCategoryID = RiskCategoryMobileID()
	UpdateItemStatus(ms.DB, f.Items[0], api.ItemCoverageStatusApproved, "")

	yesterday := time.Now().UTC().Add(-24 * time.Hour).Truncate(24 * time.Hour)

	tests := []struct {
		name           string
		policy         Policy
		riskCategoryID uuid.UUID
		amount         api.Currency
	}{
		{
			name:           "mobile",
			policy:         f.Policies[0],
			riskCategoryID: RiskCategoryMobileID(),
			amount:         1000,
		},
		{
			name:           "stationary",
			policy:         f.Policies[0],
			riskCategoryID: RiskCategoryStationaryID(),
			amount:         2000,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			ms.NoError(tt.policy.CreateRenewalLedgerEntry(ms.DB, tt.riskCategoryID, tt.amount))

			var rc RiskCategory
			Must(rc.FindByID(ms.DB, tt.riskCategoryID))

			var l LedgerEntries
			Must(ms.DB.Where("risk_category_name = ?", rc.Name).All(&l))
			ms.Equal(1, len(l))
			ms.Equal(-tt.amount, l[0].Amount)
			ms.Equal(LedgerEntryTypeCoverageRenewal, l[0].Type)
			ms.Equal(tt.policy.ID, l[0].PolicyID)
			ms.Equal(yesterday, l[0].DateSubmitted)
		})
	}
}

func (ms *ModelSuite) Test_ImportPolicies() {
	vehicleCategory := CreateCategoryFixtures(ms.DB, 1).ItemCategories[0]
	vehicleCategory.RiskCategoryID = riskCategoryVehicleID
	Must(ms.DB.Update(&vehicleCategory))

	createHouseholdEntity(ms.DB)

	file := strings.NewReader(`Account_Number,Veh_Year,Veh_Make,Veh_Model,Coverage_Id,Vehicle_Id,Veh_VIN,Covered_Value,Monthly_Charge,Start_Date,End_Date,Country_Code_Id,NAMECUST,Country_Description
200014,2001,TOYOTA,TOWNACE NOAH,18191,7979,4776,8200,15,6/8/2018 11:49,NULL,PG,"STEWART, JIMMY                                               ",Papua New Guinea`)

	got, err := ImportPolicies(ms.DB, file)
	ms.NoError(err)
	want := api.PoliciesImportResponse{
		LinesProcessed:  1,
		PoliciesCreated: 1,
		ItemsCreated:    1,
	}
	ms.Equal(want, got)

	var newPolicy Policy
	ms.NoError(ms.DB.Where("household_id = ?", "200014").First(&newPolicy))
}

func (ms *ModelSuite) Test_importPolicy() {
	catID := CreateCategoryFixtures(ms.DB, 1).ItemCategories[0].ID
	policy := CreatePolicyFixtures(ms.DB, FixturesConfig{}).Policies[0]

	teamEntityCode := CreateEntityFixture(ms.DB).Code

	newHousehold := map[string]string{
		"Account_Number":      "123456",
		"Veh_Year":            "2001",
		"Veh_Make":            "Toyota",
		"Veh_Model":           "Camry",
		"Coverage_Id":         "1",
		"Vehicle_Id":          "2",
		"Veh_VIN":             "JT4RN56S0F0075837",
		"Covered_Value":       "8200",
		"Monthly_Charge":      "15",
		"Start_Date":          "6/8/2018 11:49",
		"End_Date":            "NULL",
		"Country_Code_Id":     "US",
		"NAMECUST":            "Smith, John",
		"Country_Description": "United States",
	}
	existingHousehold := map[string]string{
		"Account_Number":      policy.HouseholdID.String,
		"Veh_Year":            "2010",
		"Veh_Make":            "Honda",
		"Veh_Model":           "Civic",
		"Coverage_Id":         "3",
		"Vehicle_Id":          "4",
		"Veh_VIN":             "2HGES15361H903843",
		"Covered_Value":       "4500",
		"Monthly_Charge":      "10",
		"Start_Date":          "11/11/2011 11:11",
		"End_Date":            "NULL",
		"Country_Code_Id":     "CA",
		"NAMECUST":            "Jameson, Rick",
		"Country_Description": "Canada",
	}
	newTeam := map[string]string{
		"Veh_Make":          "Nissan",
		"Veh_Model":         "Versa",
		"Veh_Year":          "2016",
		"Veh_VIN":           "1111",
		"Person":            "Bono",
		"Covered_Value":     "14500",
		"Statement Name":    "2016 Nissan Versa Bono",
		"Policy Name":       "XYZ Policy",
		"Entity":            teamEntityCode,
		"Cost Center":       "SVEH12",
		"Account":           "63500",
		"Ledger Entry Desc": "Bono Vehicle",
	}

	now := time.Now().UTC()

	tests := []struct {
		name       string
		data       map[string]string
		wantErr    string
		wantPolicy Policy
		wantItem   Item
		wantPerson PolicyDependent
	}{
		{
			name: "create household policy and item",
			data: newHousehold,
			wantPolicy: Policy{
				Name:         "Smith, John household",
				Type:         api.PolicyTypeHousehold,
				EntityCodeID: householdEntityID,
				HouseholdID:  nulls.NewString("123456"),
			},
			wantItem: Item{
				Name:              "2001 Toyota Camry",
				Country:           "United States",
				Make:              "Toyota",
				Model:             "Camry",
				SerialNumber:      "JT4RN56S0F0075837",
				CoverageAmount:    8200 * domain.CurrencyFactor,
				CoverageStartDate: time.Date(2018, 6, 8, 0, 0, 0, 0, time.UTC),
				Year:              nulls.NewInt(2001),
			},
		},
		{
			name: "create item only",
			data: existingHousehold,
			wantItem: Item{
				Name:              "2010 Honda Civic",
				Country:           "Canada",
				Make:              "Honda",
				Model:             "Civic",
				SerialNumber:      "2HGES15361H903843",
				CoverageAmount:    4500 * domain.CurrencyFactor,
				CoverageStartDate: time.Date(2011, 11, 11, 0, 0, 0, 0, time.UTC),
				Year:              nulls.NewInt(2010),
			},
		},
		{
			name: "create team policy and item",
			data: newTeam,
			wantPolicy: Policy{
				Name:          "XYZ Policy",
				Type:          api.PolicyTypeTeam,
				EntityCodeID:  EntityCodeID(teamEntityCode),
				CostCenter:    "SVEH12",
				Account:       "63500",
				AccountDetail: "Bono Vehicle",
			},
			wantItem: Item{
				Name:              "2016 Nissan Versa Bono",
				Make:              "Nissan",
				Model:             "Versa",
				Year:              nulls.NewInt(2016),
				SerialNumber:      "1111",
				CoverageAmount:    14500 * domain.CurrencyFactor,
				CoverageStartDate: time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC),
			},
			wantPerson: PolicyDependent{
				Name: "Bono",
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			_, _, err := importPolicy(ms.DB, tt.data, catID, now)
			if tt.wantErr != "" {
				ms.Error(err)
				ms.Contains(err.Error(), tt.wantErr)
				return
			}

			ms.NoError(err)

			var p Policy
			if tt.wantPolicy.Name == "" {
				p = policy
			} else {
				ms.NoError(ms.DB.Where("name = ?", tt.wantPolicy.Name).First(&p))
				ms.Equal(tt.wantPolicy.Name, p.Name)
				ms.Equal(tt.wantPolicy.Type, p.Type)
				ms.Equal(tt.wantPolicy.HouseholdID, p.HouseholdID)
				ms.Equal(tt.wantPolicy.EntityCodeID, p.EntityCodeID)
				ms.Equal(tt.wantPolicy.CostCenter, p.CostCenter)
				ms.Equal(tt.wantPolicy.Account, p.Account)
				ms.Equal(tt.wantPolicy.AccountDetail, p.AccountDetail)
			}

			var i Item
			ms.NoError(ms.DB.Order("created_at DESC").First(&i))
			ms.Equal(tt.wantItem.Name, i.Name)
			ms.Equal(catID, i.CategoryID)
			ms.Equal(tt.wantItem.Country, i.Country)
			ms.Equal(p.ID, i.PolicyID)
			ms.Equal(tt.wantItem.Make, i.Make)
			ms.Equal(tt.wantItem.Model, i.Model)
			ms.Equal(tt.wantItem.SerialNumber, i.SerialNumber)
			ms.Equal(tt.wantItem.CoverageAmount, i.CoverageAmount)
			ms.Equal(api.ItemCoverageStatusApproved, i.CoverageStatus)
			ms.Equal(tt.wantItem.CoverageStartDate, i.CoverageStartDate)
			ms.Equal(riskCategoryVehicleID, i.RiskCategoryID)
			ms.Equal(tt.wantItem.Year, i.Year)
			ms.Equal(domain.EndOfMonth(now).AddDate(0, -1, 0), i.PaidThroughDate)

			if tt.wantPerson.Name != "" {
				var pd PolicyDependent
				ms.NoError(pd.FindByName(ms.DB, p.ID, tt.wantPerson.Name))
			}
		})
	}
}

func (ms *ModelSuite) Test_parseCoveredValue() {
	tests := []struct {
		s       string
		want    int
		wantErr bool
	}{
		{
			s:       "",
			wantErr: true,
		},
		{
			s:       "x",
			wantErr: true,
		},
		{
			s:       "-1",
			wantErr: true,
		},
		{
			s:       "0",
			wantErr: true,
		},
		{
			s:    "1",
			want: 100,
		},
		{
			s:    "100",
			want: 10000,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.s, func(t *testing.T) {
			got, err := parseCoveredValue(tt.s)
			if tt.wantErr {
				ms.Error(err)
				return
			}

			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) Test_parseVehicleYear() {
	tests := []struct {
		s       string
		want    int
		wantErr bool
	}{
		{
			s:       "",
			wantErr: true,
		},
		{
			s:       "x",
			wantErr: true,
		},
		{
			s:       "-1",
			wantErr: true,
		},
		{
			s:       "100",
			wantErr: true,
		},
		{
			s:       "1910",
			wantErr: true,
		},
		{
			s:       "2051",
			wantErr: true,
		},
		{
			s:    "50",
			want: 1950,
		},
		{
			s:    "99",
			want: 1999,
		},
		{
			s:    "0",
			want: 2000,
		},
		{
			s:    "49",
			want: 2049,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.s, func(t *testing.T) {
			got, err := parseVehicleYear(tt.s)
			if tt.wantErr {
				ms.Error(err)
				return
			}

			ms.Equal(tt.want, got)
		})
	}
}
