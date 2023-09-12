package models

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestUser_Validate() {
	t := ms.T()
	tests := []struct {
		name     string
		user     User
		wantErr  bool
		errField string
	}{
		{
			name: "minimum",
			user: User{
				Email:   "user@example.com",
				AppRole: AppRoleCustomer,
			},
			wantErr: false,
		},
		{
			name: "missing email",
			user: User{
				AppRole: AppRoleCustomer,
			},
			wantErr:  true,
			errField: "User.Email",
		},
		{
			name: "missing approle",
			user: User{
				Email: "dummy@dusos.com",
			},
			wantErr:  true,
			errField: "User.AppRole",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.user.Validate(DB)
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

func (ms *ModelSuite) TestUser_CreateInitialPolicy() {
	t := ms.T()

	policyCount := 1
	pf := CreatePolicyFixtures(ms.DB, FixturesConfig{NumberOfPolicies: policyCount})
	policy := pf.Policies[0]

	uf := CreateUserFixtures(ms.DB, 3)
	userNoPolicy := uf.Users[0]
	userForHouseholdID := uf.Users[1]
	userWithPolicy := uf.Users[2]

	pUser := PolicyUser{
		PolicyID: policy.ID,
		UserID:   userWithPolicy.ID,
	}
	policyUserCount := policyCount + 1

	ms.NoError(pUser.Create(ms.DB))

	tests := []struct {
		name            string
		user            User
		householdID     string
		wantErr         bool
		wantPolicies    int
		wantPolicyUsers int
	}{
		{
			name:    "missing ID",
			user:    User{},
			wantErr: true,
		},
		{
			name:            "user already has policy",
			user:            userWithPolicy,
			wantErr:         false,
			wantPolicies:    policyCount,
			wantPolicyUsers: policyUserCount,
		},
		{
			name:            "new user but has same household_id",
			user:            userForHouseholdID,
			householdID:     policy.HouseholdID.String,
			wantErr:         false,
			wantPolicies:    policyCount,
			wantPolicyUsers: policyUserCount + 1,
		},
		{
			name:            "policy to be created",
			householdID:     "otherHHID",
			user:            userNoPolicy,
			wantErr:         false,
			wantPolicies:    policyCount + 1,
			wantPolicyUsers: policyUserCount + 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.CreateInitialPolicy(DB, tt.householdID)
			if tt.wantErr {
				ms.Error(err)
				return
			}

			ms.NoError(err)

			policyUsers := PolicyUsers{}
			err = ms.DB.All(&policyUsers)
			ms.NoError(err, "error fetching policyUsers")
			ms.Len(policyUsers, tt.wantPolicyUsers, "incorrect number of policyUsers")

			policies := Policies{}
			err = ms.DB.All(&policies)
			ms.NoError(err, "error fetching policies")
			ms.Len(policies, tt.wantPolicies, "incorrect number of policies")
		})
	}
}

func (ms *ModelSuite) TestUser_FindStewards() {
	CreateUserFixtures(ms.DB, 3)
	steward0 := CreateAdminUsers(ms.DB)[AppRoleSteward]
	steward1 := CreateAdminUsers(ms.DB)[AppRoleSteward]

	var users Users
	users.FindStewards(ms.DB)
	want := map[uuid.UUID]bool{steward0.ID: true, steward1.ID: true}

	got := map[uuid.UUID]bool{}
	for _, s := range users {
		got[s.ID] = true
	}

	ms.EqualValues(want, got, "incorrect steward ids")
}

func (ms *ModelSuite) TestUser_FindSignators() {
	CreateUserFixtures(ms.DB, 3)
	signator0 := CreateAdminUsers(ms.DB)[AppRoleSignator]
	signator1 := CreateAdminUsers(ms.DB)[AppRoleSignator]

	var users Users
	users.FindSignators(ms.DB)
	want := map[uuid.UUID]bool{signator0.ID: true, signator1.ID: true}

	got := map[uuid.UUID]bool{}
	for _, s := range users {
		got[s.ID] = true
	}

	ms.EqualValues(want, got, "incorrect signator ids")
}

func (ms *ModelSuite) TestUser_EmailOfChoice() {
	justEmail := User{Email: "justemail@example.com"}
	hasOverride := User{Email: "main@example.com", EmailOverride: "override@example.com"}

	got := justEmail.EmailOfChoice()
	ms.Equal(justEmail.Email, got, "incorrect Email for user with no override email")

	got = hasOverride.EmailOfChoice()
	ms.Equal(hasOverride.EmailOverride, got, "incorrect Email for user with an override email")
}

func (ms *ModelSuite) TestUser_Name() {
	t := ms.T()
	tests := []struct {
		name string
		user User
		want string
	}{
		{
			name: "only first",
			user: User{FirstName: "  OnlyFirst "},
			want: "OnlyFirst",
		},
		{
			name: "only last",
			user: User{LastName: "  OnlyLast "},
			want: "OnlyLast",
		},
		{
			name: "no extra spaces",
			user: User{FirstName: "First", LastName: "Last"},
			want: "First Last",
		},
		{
			name: "has extra spaces",
			user: User{FirstName: "  First  ", LastName: "  Last  "},
			want: "First Last",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.Name()
			ms.Equal(tt.want, got, "incorrect user name")
		})
	}
}

func (ms *ModelSuite) TestUser_OwnsFile() {
	userFixtures := CreateUserFixtures(ms.DB, 2)
	userOwnsFile := userFixtures.Users[1]
	userNoFile := userFixtures.Users[0]

	fileFixtures := CreateFileFixtures(ms.DB, 1, userOwnsFile.ID)
	file := fileFixtures.Files[0]

	tests := []struct {
		name    string
		user    User
		file    File
		want    bool
		wantErr bool
	}{
		{
			name:    "user not valid",
			file:    file,
			wantErr: true,
		},
		{
			name:    "not owned",
			user:    userNoFile,
			file:    file,
			want:    false,
			wantErr: false,
		},
		{
			name:    "owned",
			user:    userOwnsFile,
			file:    file,
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			ownsFile, err := tt.user.OwnsFile(ms.DB, tt.file)
			if tt.wantErr {
				ms.Error(err)
				return
			}
			ms.NoError(err)
			ms.Equal(tt.want, ownsFile, "incorrect result from OwnsFile")
		})
	}
}

func (ms *ModelSuite) TestUser_ConvertToAPI() {
	f := CreatePolicyFixtures(ms.DB, FixturesConfig{})
	user := f.Users[0]

	got := user.ConvertToAPI(ms.DB, false)

	ms.Equal(user.ID, got.ID, "ID is not correct")
	ms.Equal(user.Email, got.Email, "Email is not correct")
	ms.Equal(user.EmailOverride, got.EmailOverride, "EmailOverride is not correct")
	ms.Equal(user.FirstName, got.FirstName, "FirstName is not correct")
	ms.Equal(user.LastName, got.LastName, "LastName is not correct")
	ms.Equal(user.Name(), got.Name, "Name is not correct")
	ms.Equal(string(user.AppRole), got.AppRole, "AppRole is not correct")
	ms.Equal(user.LastLoginUTC, got.LastLoginUTC, "LastLoginUTC is not correct")
	ms.Equal(user.Country, got.Country, "Country is not correct")
	ms.Equal(user.CountryCode, got.CountryCode, "CountryCode is not correct")
	ms.EqualNullUUID(user.PhotoFileID, got.PhotoFileID, "PhotoFileID is not correct")

	ms.Equal(0, len(got.Policies), "Policies should not be hydrated")

	got = user.ConvertToAPI(ms.DB, true)

	ms.Greater(len(user.Policies), 0, "test should be revised, fixture has no Policies")
	ms.Equal(len(got.Policies), len(user.Policies), "Policies is not correct length")
}

func (ms *ModelSuite) TestConvertToPolicyMember() {
	user := User{
		ID:            domain.GetUUID(),
		Email:         randStr(10),
		EmailOverride: randStr(10),
		FirstName:     randStr(10),
		LastName:      randStr(10),
		Country:       randStr(10),
		LastLoginUTC:  time.Now(),
	}
	polUserID := domain.GetUUID()
	got := user.ConvertToPolicyMember(polUserID)

	ms.Equal(user.ID, got.ID, "ID is not correct")
	ms.Equal(user.FirstName, got.FirstName, "FirstName is not correct")
	ms.Equal(user.LastName, got.LastName, "LastName is not correct")
	ms.Equal(user.Email, got.Email, "Email is not correct")
	ms.Equal(user.EmailOverride, got.EmailOverride, "EmailOverride is not correct")
	ms.Equal(user.LastLoginUTC, got.LastLoginUTC, "LastLoginUTC is not correct")
	ms.Equal(user.Country, got.Country, "Country is not correct")
	ms.Equal(polUserID, got.PolicyUserID, "PolicyUserID is not correct")
}
