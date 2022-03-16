package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_PoliciesList() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreatePolicyFixtures(as.DB, fixConfig)

	fixtures.Users[0].FirstName = "John"
	fixtures.Users[0].LastName = "Bunyan"
	as.NoError(as.DB.Update(&fixtures.Users[0]))

	for _, p := range fixtures.Policies {
		p.LoadMembers(as.DB, false)
		p.LoadDependents(as.DB, false)
	}

	normalUser := fixtures.Policies[1].Members[0]
	appAdmin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name          string
		actor         models.User
		queryString   string
		wantCount     int
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "unauthenticated",
			actor:         models.User{},
			wantCount:     0,
			wantStatus:    http.StatusUnauthorized,
			wantInBody:    "",
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
		{
			name:          "admin",
			actor:         appAdmin,
			wantCount:     len(fixtures.Policies),
			wantStatus:    http.StatusOK,
			wantInBody:    fixtures.Policies[1].ID.String(),
			notWantInBody: "",
		},
		{
			name:        "admin with search",
			actor:       appAdmin,
			queryString: "?limit=1&search=john",
			wantCount:   1,
			wantStatus:  http.StatusOK,
		},
		{
			name:        "admin with username search",
			actor:       appAdmin,
			queryString: "?search=john bunyan",
			wantCount:   1,
			wantStatus:  http.StatusOK,
		},
		{
			name:          "user",
			actor:         normalUser,
			wantCount:     1,
			wantStatus:    http.StatusOK,
			wantInBody:    fixtures.Policies[1].ID.String(),
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies" + tt.queryString)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				as.Contains(body, tt.wantInBody)
			}
			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}

			var response struct {
				Meta api.Meta     `json:"meta"`
				Data api.Policies `json:"data"`
			}
			dec := json.NewDecoder(strings.NewReader(body))
			dec.DisallowUnknownFields()
			err := dec.Decode(&response)

			as.NoError(err)
			as.Equal(tt.wantCount, len(response.Data))
		})
	}
}

func (as *ActionSuite) Test_PoliciesView() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreatePolicyFixtures(as.DB, fixConfig)

	for _, p := range fixtures.Policies {
		p.LoadMembers(as.DB, false)
		p.LoadDependents(as.DB, false)
	}

	appAdmin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name          string
		actor         models.User
		policyID      uuid.UUID
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "unauthenticated",
			actor:         models.User{},
			policyID:      fixtures.Policies[1].ID,
			wantStatus:    http.StatusUnauthorized,
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
		{
			name:          "non-policy user",
			actor:         fixtures.Policies[1].Members[0],
			policyID:      fixtures.Policies[0].ID,
			wantStatus:    http.StatusNotFound,
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
		{
			name:       "policy user",
			actor:      fixtures.Policies[0].Members[0],
			policyID:   fixtures.Policies[0].ID,
			wantStatus: http.StatusOK,
			wantInBody: fixtures.Policies[0].ID.String(),
		},
		{
			name:          "admin",
			actor:         appAdmin,
			policyID:      fixtures.Policies[1].ID,
			wantStatus:    http.StatusOK,
			wantInBody:    fixtures.Policies[1].ID.String(),
			notWantInBody: "",
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies/" + tt.policyID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				as.Contains(body, tt.wantInBody)
			}
			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}
			var policy api.Policy
			dec := json.NewDecoder(strings.NewReader(body))
			dec.DisallowUnknownFields()
			err := dec.Decode(&policy)
			as.NoError(err)
			as.Equal(tt.policyID, policy.ID)
			as.Equal(1, len(policy.Members))
		})
	}
}

func (as *ActionSuite) Test_PoliciesCreateTeam() {
	fixtures := models.CreatePolicyFixtures(as.DB, models.FixturesConfig{NumberOfEntityCodes: 1})

	entCode := fixtures.EntityCodes[0]
	user := fixtures.Policies[0].Members[0]

	goodPolicy := api.PolicyCreate{
		Name:       "my policy",
		CostCenter: "abc123",
		Account:    "def456",
		EntityCode: entCode.Code,
	}

	policyMissingCC := goodPolicy
	policyMissingCC.CostCenter = ""

	tests := []struct {
		name          string
		actor         models.User
		input         api.PolicyCreate
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "not authenticated",
			actor:         models.User{},
			input:         goodPolicy,
			wantStatus:    http.StatusUnauthorized,
			wantInBody:    api.ErrorNotAuthorized.String(),
			notWantInBody: goodPolicy.CostCenter,
		},
		{
			name:          "missing Cost Center",
			actor:         user,
			input:         policyMissingCC,
			wantStatus:    http.StatusBadRequest,
			notWantInBody: policyMissingCC.Account,
		},
		{
			name:       "good input",
			actor:      user,
			input:      goodPolicy,
			wantStatus: http.StatusOK,
			wantInBody: goodPolicy.CostCenter,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Post(tt.input)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				as.Contains(body, tt.wantInBody)
			}
			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}
			var policy api.Policy
			as.NoError(json.Unmarshal([]byte(body), &policy))
			as.Equal(tt.input.CostCenter, policy.CostCenter)
			as.Equal(tt.input.Account, policy.Account)
			as.Equal(tt.input.EntityCode, policy.EntityCode.Code)
			as.Equal(api.PolicyTypeTeam, policy.Type)
		})
	}
}

func (as *ActionSuite) Test_PoliciesUpdate() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreatePolicyFixtures(as.DB, fixConfig)

	for _, p := range fixtures.Policies {
		p.LoadMembers(as.DB, false)
		p.LoadDependents(as.DB, false)
	}

	// alias a couple users
	appAdmin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]
	normalUser := fixtures.Policies[1].Members[0]

	id1 := "654978"
	id2 := "09876"
	id3 := "998877"

	tests := []struct {
		name          string
		actor         models.User
		policy        models.Policy
		update        api.PolicyUpdate
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:   "valid household id",
			actor:  normalUser,
			policy: fixtures.Policies[1],
			update: api.PolicyUpdate{
				Name:        fixtures.Policies[1].Name,
				HouseholdID: &id1,
			},
			wantStatus: http.StatusOK,
			wantInBody: fixtures.Policies[1].ID.String(),
		},
		{
			name:   "other person's policy",
			actor:  normalUser,
			policy: fixtures.Policies[0],
			update: api.PolicyUpdate{
				Name:        fixtures.Policies[1].Name,
				HouseholdID: &id2,
			},
			wantStatus:    http.StatusNotFound,
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
		{
			name:   "admin update other person's policy",
			actor:  appAdmin,
			policy: fixtures.Policies[1],
			update: api.PolicyUpdate{
				Name:        fixtures.Policies[1].Name,
				HouseholdID: &id3,
			},
			wantStatus: http.StatusOK,
			wantInBody: fixtures.Policies[1].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies/" + tt.policy.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Put(tt.update)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				as.Contains(body, tt.wantInBody)
			}
			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}
			var policy api.Policy
			as.NoError(json.Unmarshal([]byte(body), &policy))
			as.Equal(*tt.update.HouseholdID, policy.HouseholdID)
		})
	}
}

func (as *ActionSuite) Test_PoliciesListMembers() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreatePolicyFixtures(as.DB, fixConfig)

	for _, p := range fixtures.Policies {
		p.LoadMembers(as.DB, false)
		p.LoadDependents(as.DB, false)
	}

	// alias a couple users
	appAdmin := fixtures.Policies[0].Members[0]
	normalUser := fixtures.Policies[1].Members[0]

	// change user 0 to an admin
	appAdmin.AppRole = models.AppRoleSteward
	err := appAdmin.Update(as.DB)
	as.NoError(err, "failed to make first policy user an app admin")

	tests := []struct {
		name          string
		actor         models.User
		policyID      string
		wantCount     int
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "unauthenticated",
			actor:         models.User{},
			policyID:      fixtures.Policies[0].ID.String(),
			wantCount:     0,
			wantStatus:    http.StatusUnauthorized,
			wantInBody:    "",
			notWantInBody: fixtures.Policies[0].Members[0].ID.String(),
		},
		{
			name:          "admin",
			actor:         appAdmin,
			policyID:      fixtures.Policies[0].ID.String(),
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusOK,
			wantInBody:    appAdmin.ID.String(),
			notWantInBody: normalUser.ID.String(),
		},
		{
			name:          "admin - other user's policy",
			actor:         appAdmin,
			policyID:      fixtures.Policies[1].ID.String(),
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusOK,
			wantInBody:    normalUser.ID.String(),
			notWantInBody: appAdmin.ID.String(),
		},
		{
			name:          "user",
			actor:         normalUser,
			policyID:      fixtures.Policies[1].ID.String(),
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusOK,
			wantInBody:    normalUser.ID.String(),
			notWantInBody: appAdmin.ID.String(),
		},
		{
			name:          "normal user, other user's policy",
			actor:         normalUser,
			policyID:      fixtures.Policies[0].ID.String(),
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusNotFound,
			wantInBody:    "",
			notWantInBody: appAdmin.ID.String(),
		},
		{
			name:          "invalid ID",
			actor:         normalUser,
			policyID:      "abc123",
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusNotFound,
			wantInBody:    "",
			notWantInBody: normalUser.ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies/" + tt.policyID + "/members")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				as.Contains(body, tt.wantInBody)
			}
			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}
			var members api.PolicyMembers
			err := json.Unmarshal([]byte(body), &members)
			as.NoError(err)
			as.Equal(tt.wantCount, len(members))
		})
	}
}

func (as *ActionSuite) Test_PoliciesInviteMember() {
	fixConfig := models.FixturesConfig{
		NumberOfEntityCodes: 2,
		NumberOfPolicies:    2,
		UsersPerPolicy:      1,
	}

	teamPolicies := models.CreateTeamPolicyFixtures(as.DB, fixConfig).Policies
	team0member0 := teamPolicies[0].Members[0]
	team1member0 := teamPolicies[1].Members[0]

	housePolicies := models.CreatePolicyFixtures(as.DB, fixConfig).Policies
	house0member0 := housePolicies[0].Members[0]
	house1member0 := housePolicies[1].Members[0]

	tests := []struct {
		name               string
		policyID           uuid.UUID
		actor              models.User
		inviteeEmail       string
		inviteeName        string
		wantStatus         int
		wantEventTriggered bool
	}{
		{
			name:               "existing team policy member, no event",
			policyID:           teamPolicies[0].ID,
			actor:              team0member0,
			inviteeEmail:       team0member0.Email,
			wantStatus:         http.StatusNoContent,
			wantEventTriggered: false,
		},
		{
			name:               "existing user for team policy, not policy member, no event",
			policyID:           teamPolicies[0].ID,
			actor:              team0member0,
			inviteeEmail:       team1member0.Email,
			wantStatus:         http.StatusNoContent,
			wantEventTriggered: false,
		},
		{
			name:               "new user for team policy",
			policyID:           teamPolicies[0].ID,
			actor:              team1member0,
			inviteeEmail:       "new-team-user-testing@invites-r-us.com",
			inviteeName:        "New TeamUser",
			wantStatus:         http.StatusNoContent,
			wantEventTriggered: true,
		},
		{
			name:               "existing household policy member, no event",
			policyID:           housePolicies[0].ID,
			actor:              house0member0,
			inviteeEmail:       house0member0.Email,
			wantStatus:         http.StatusBadRequest,
			wantEventTriggered: false,
		},
		{
			name:               "existing user for household policy, not policy member, no event",
			policyID:           housePolicies[0].ID,
			actor:              house0member0,
			inviteeEmail:       house1member0.Email,
			wantStatus:         http.StatusBadRequest,
			wantEventTriggered: false,
		},
		{
			name:               "new user for household policy",
			policyID:           housePolicies[1].ID,
			actor:              house1member0,
			inviteeEmail:       "new-house-user-testing@invites-r-us.com",
			inviteeName:        "New HouseUser",
			wantStatus:         http.StatusNoContent,
			wantEventTriggered: true,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			createInviteEventDetected := false
			deleteFn1, err := models.RegisterEventDetector(domain.EventApiPolicyUserInviteCreated, &createInviteEventDetected)
			as.NoError(err)
			defer deleteFn1()

			input := api.PolicyUserInviteCreate{
				Email: tt.inviteeEmail,
				Name:  tt.inviteeName,
			}

			req := as.JSON("/policies/" + tt.policyID.String() + "/members")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Post(input)

			as.Equal(tt.wantStatus, res.Code, "http status code not as expected")
			as.Equal(tt.wantEventTriggered, createInviteEventDetected, "event detection not as expected")
		})
	}
}

func (as *ActionSuite) Test_PolicyLedgerReportCreate() {

	f0 := models.CreatePolicyFixtures(as.DB, models.FixturesConfig{})
	otherUser := f0.Users[0]

	f := as.createFixturesForLedger()
	policy := f.Policies[0]
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	now := time.Now().UTC()
	nowMonth := int(now.Month())
	nowYear := now.Year()

	tests := []struct {
		name       string
		actor      models.User
		month      int
		year       int
		reportType string
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      otherUser,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "monthly report",
			actor:      normalUser,
			month:      nowMonth,
			year:       nowYear,
			reportType: models.ReportTypeMonthly,
			wantStatus: http.StatusOK,
		},
		{
			name:       "annual report",
			actor:      stewardUser,
			month:      0,
			year:       nowYear,
			reportType: models.ReportTypeAnnual,
			wantStatus: http.StatusOK,
		},
		{
			name:       "no ledger entries",
			actor:      normalUser,
			month:      nowMonth,
			year:       1972,
			reportType: models.ReportTypeMonthly,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("%s/%s/ledger-reports", policiesPath, policy.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Post(api.PolicyLedgerReportCreateInput{
				Month: tt.month,
				Year:  tt.year,
				Type:  tt.reportType,
			})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}

			if res.Code != http.StatusOK {
				return
			}

			var report api.LedgerReport
			as.NoError(json.Unmarshal([]byte(body), &report))
			as.Equal(tt.reportType, report.Type)
		})
	}
}

func (as *ActionSuite) Test_PolicyStrikesCreate() {

	f := models.CreatePolicyFixtures(as.DB, models.FixturesConfig{NumberOfPolicies: 2})
	policyOneStrike := f.Policies[0]
	policyNoStrikes := f.Policies[1]
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	strikes := models.CreateStrikeFixtures(as.DB, f.Policies, [][]*time.Time{{nil}})

	tests := []struct {
		name       string
		actor      models.User
		policy     models.Policy
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			policy:     policyOneStrike,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			policy:     policyOneStrike,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "single strike",
			actor:      stewardUser,
			policy:     policyNoStrikes,
			wantStatus: http.StatusOK,
			wantInBody: []string{`"description":"New Strike`},
		},
		{
			name:       "with old strike",
			actor:      stewardUser,
			policy:     policyOneStrike,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"description":"` + strikes[0].Description,
				`"description":"New Strike`,
				`"policy_id":"` + policyOneStrike.ID.String(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("%s/%s/%s", policiesPath, tt.policy.ID.String(), api.ResourceStrikes)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Post(api.StrikeInput{Description: "New Strike"})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}
		})
	}
}
