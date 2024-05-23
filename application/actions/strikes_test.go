package actions

import (
	"net/http"
	"testing"
	"time"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_StrikesUpdate() {

	f := models.CreatePolicyFixtures(as.DB, models.FixturesConfig{NumberOfPolicies: 2})
	policyTwoStrikes := f.Policies[0]
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	strikes := models.CreateStrikeFixtures(as.DB, f.Policies, [][]*time.Time{{nil, nil}})
	newDescription := "Updated Strike"

	tests := []struct {
		name       string
		actor      models.User
		strike     models.Strike
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			strike:     strikes[0],
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			strike:     strikes[0],
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "good",
			actor:      stewardUser,
			strike:     strikes[0],
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"id":"` + strikes[0].ID.String(),
				`"description":"` + newDescription,
				`"policy_id":"` + policyTwoStrikes.ID.String(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			uat, err := tt.actor.CreateAccessToken(as.DB)
			as.NoError(err)
			as.Session.Set(AccessTokenSessionKey, uat.AccessToken)
			req := as.JSON("%s/%s", strikesPath, tt.strike.ID.String())
			res := req.Put(api.StrikeInput{Description: newDescription})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}
		})
	}
}

func (as *ActionSuite) Test_StrikesDelete() {

	f := models.CreatePolicyFixtures(as.DB, models.FixturesConfig{NumberOfPolicies: 2})
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	strikes := models.CreateStrikeFixtures(as.DB, f.Policies, [][]*time.Time{{nil, nil}})

	tests := []struct {
		name       string
		actor      models.User
		strike     models.Strike
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			strike:     strikes[0],
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			strike:     strikes[0],
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "good",
			actor:      stewardUser,
			strike:     strikes[0],
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			uat, err := tt.actor.CreateAccessToken(as.DB)
			as.NoError(err)
			as.Session.Set(AccessTokenSessionKey, uat.AccessToken)
			req := as.JSON("%s/%s", strikesPath, tt.strike.ID.String())
			res := req.Delete()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			if tt.wantStatus == http.StatusNoContent {
				return
			}

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}
		})
	}
}
