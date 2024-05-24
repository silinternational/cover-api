package actions

import (
	"net/http"
	"testing"
	"time"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_repairsRun() {
	user := models.CreateUserFixtures(as.DB, 1).Users[0]
	admin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]
	date := time.Now().UTC().Format(domain.DateFormat)

	tests := []struct {
		name       string
		actor      models.User
		input      api.RepairRunInput
		wantStatus int
		wantError  api.ErrorKey
	}{
		{
			name:       "Unauthorized (401)",
			actor:      models.User{},
			input:      api.RepairRunInput{Date: date, RepairType: api.RepairTypeRenewal},
			wantStatus: http.StatusUnauthorized,
			wantError:  api.ErrorNotAuthorized,
		},
		{
			name:       "normal users can't do this",
			actor:      user,
			input:      api.RepairRunInput{Date: date, RepairType: api.RepairTypeRenewal},
			wantStatus: http.StatusNotFound,
			wantError:  api.ErrorNotAuthorized,
		},
		{
			name:       "invalid date format",
			actor:      admin,
			input:      api.RepairRunInput{Date: time.Now().String(), RepairType: api.RepairTypeRenewal},
			wantStatus: http.StatusBadRequest,
			wantError:  api.ErrorInvalidDate,
		},
		{
			name:       "admins can do this",
			actor:      admin,
			input:      api.RepairRunInput{Date: date, RepairType: api.RepairTypeRenewal},
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			as.SetAccessToken(tt.actor)
			req := as.JSON(repairsPath)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Post(tt.input)
			body := res.Body.Bytes()

			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned: %d\n%s", res.Code, body)
			if tt.wantStatus != http.StatusOK {
				var err api.AppError
				as.NoError(as.decodeBody(body, &err), "response data is not as expected")
				as.Equal(tt.wantError, err.Key, "error key is incorrect")
			} else {
				var repairResult api.RepairResult
				as.NoError(as.decodeBody(body, &repairResult), "response data is not as expected")
				as.Equal(tt.input.RepairType, repairResult.RepairType, "repair_type is incorrect")
			}
		})
	}
}
