package actions

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_AuthLogin_Invite() {
	db := as.DB
	_ = models.CreateUserFixtures(db, 1)
	// users := userFixtures.Users

	missingCode := "7cc06a0f-00a6-4b2d-9cd6-c0820ae46662"

	invite := models.CreatePolicyUserInviteFixtures(db, models.Policies{}, 2).PolicyUserInvites[0]

	tests := []struct {
		name         string
		queryParams  string
		wantStatus   int
		wantContains string
		wantCode     string
	}{
		{
			name: "Bad Invite Code",
			queryParams: fmt.Sprintf("%s=123456&%s=badInviteCode",
				ClientIDParam, InviteCodeParam),
			wantStatus:   http.StatusBadRequest,
			wantContains: string(api.ErrorProcessingAuthInviteCode),
		},
		{
			name: "Invite Code not in DB",
			queryParams: fmt.Sprintf("%s=123456&%s=%v",
				ClientIDParam, InviteCodeParam, missingCode),
			wantStatus:   http.StatusNotFound,
			wantContains: string(api.ErrorProcessingAuthInviteCode),
		},
		{
			name: "All Good",
			queryParams: fmt.Sprintf("%s=123456&%s=%v",
				ClientIDParam, InviteCodeParam, invite.ID),
			wantStatus:   http.StatusOK,
			wantContains: `"RedirectURL":"` + domain.Env.SamlSsoURL,
			wantCode:     invite.ID.String(),
		},
	}
	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			path := "/auth/login?" + tt.queryParams
			body, status := as.request("POST", path, "", nil)

			as.Equal(tt.wantStatus, status, "incorrect status code returned, body: %s", body)
			as.Contains(string(body), tt.wantContains, "incorrect response body")

			if tt.wantCode == "" {
				return
			}

			// FIXME
			// sessInviteCode := as.Session.Get(InviteCodeSessionKey)
			// as.Equal(tt.wantCode, sessInviteCode)
		})
	}
}
