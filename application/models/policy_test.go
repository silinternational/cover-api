package models

import (
	"fmt"
	"testing"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
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
				Type: "invalid",
			},
			wantErr:  true,
			errField: "Policy.Type",
		},
		{
			name:     "missing type",
			Policy:   Policy{},
			wantErr:  true,
			errField: "Policy.Type",
		},
		{
			name: "valid type, missing household id",
			Policy: Policy{
				Type: api.PolicyTypeHousehold,
			},
			wantErr:  true,
			errField: "Policy.HouseholdID",
		},
		{
			name: "valid type",
			Policy: Policy{
				Type:        api.PolicyTypeHousehold,
				HouseholdID: nulls.NewString("123456"),
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

func (ms *ModelSuite) TestPolicy_LoadMembers() {
	rando := randStr(6)
	policy := Policy{
		Type:        api.PolicyTypeHousehold,
		HouseholdID: nulls.NewString(rando),
	}
	MustCreate(ms.DB, &policy)

	user := User{
		Email:     rando + "@testerson.com",
		FirstName: "Test",
		LastName:  "Testerson",
		IsBlocked: false,
		StaffID:   rando,
		AppRole:   AppRoleUser,
	}
	MustCreate(ms.DB, &user)

	pu := PolicyUser{
		PolicyID: policy.ID,
		UserID:   user.ID,
	}
	MustCreate(ms.DB, &pu)

	policy.LoadMembers(ms.DB, false)
	ms.Len(policy.Members, 1)
}

func (ms *ModelSuite) TestPolicy_LoadDependents() {
	rando := randStr(6)
	policy := Policy{
		Type:        api.PolicyTypeHousehold,
		HouseholdID: nulls.NewString(rando),
	}
	MustCreate(ms.DB, &policy)

	user := User{
		Email:     rando + "@testerson.com",
		FirstName: "Test",
		LastName:  "Testerson",
		IsBlocked: false,
		StaffID:   rando,
		AppRole:   AppRoleUser,
	}
	MustCreate(ms.DB, &user)

	pu := PolicyDependent{
		PolicyID:       policy.ID,
		Name:           rando + "-kiddo",
		Relationship:   api.PolicyDependentRelationshipChild,
		Location:       "Bahamas",
		ChildBirthYear: 2000,
	}
	MustCreate(ms.DB, &pu)

	policy.LoadDependents(ms.DB, false)
	ms.Len(policy.Dependents, 1)
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
			items[i] = UpdateItemStatus(ms.DB, items[i], api.ItemCoverageStatusApproved)
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
	e := EntityCode{
		Code: randStr(3),
		Name: "Acme, Inc.",
	}
	MustCreate(ms.DB, &e)

	oldPolicy := Policy{
		Type:         api.PolicyTypeCorporate,
		HouseholdID:  nulls.NewString("abc123"),
		CostCenter:   "xyz789",
		Account:      "123457890",
		EntityCodeID: nulls.NewUUID(e.ID),
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
					OldValue:  oldPolicy.EntityCodeID.UUID.String(),
					NewValue:  newPolicy.EntityCodeID.UUID.String(),
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
				Description: fmt.Sprintf(`field HouseholdID changed from "%s" to "%s" by %s`,
					policy.HouseholdID.String, newHouseholdID, user.Name()),
				OldValue: policy.HouseholdID.String,
				NewValue: newHouseholdID,
			},
		},
		{
			name:   "EntityCodeID",
			policy: policy,
			user:   user,
			update: FieldUpdate{
				FieldName: "EntityCodeID",
				OldValue:  policy.EntityCodeID.UUID.String(),
				NewValue:  newEntityCodeID,
			},
			want: PolicyHistory{
				PolicyID:  policy.ID,
				UserID:    user.ID,
				Action:    api.HistoryActionUpdate,
				FieldName: "EntityCodeID",
				Description: fmt.Sprintf(`field EntityCodeID changed from "%s" to "%s" by %s`,
					policy.EntityCodeID.UUID.String(), newEntityCodeID, user.Name()),
				OldValue: policy.EntityCodeID.UUID.String(),
				NewValue: newEntityCodeID,
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.policy.NewHistory(CreateTestContext(tt.user), api.HistoryActionUpdate, tt.update)
			ms.False(tt.want.NewValue == tt.want.OldValue, "test isn't correctly checking a field update")
			ms.Equal(tt.want.PolicyID, got.PolicyID, "PolicyID is not correct")
			ms.Equal(tt.want.UserID, got.UserID, "FieldName is not correct")
			ms.Equal(tt.want.Action, got.Action, "Action is not correct")
			ms.Equal(tt.want.FieldName, got.FieldName, "FieldName is not correct")
			ms.Equal(tt.want.Description, got.Description, "Description is not correct")
			ms.Equal(tt.want.OldValue, got.OldValue, "OldValue is not correct")
			ms.Equal(tt.want.NewValue, got.NewValue, "NewValue is not correct")
		})
	}
}
