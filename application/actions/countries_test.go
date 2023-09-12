package actions

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_countriesByCode() {
	actor := models.CreateUserFixtures(as.DB, 1).Users[0]

	tests := []struct {
		name       string
		actor      models.User
		code       string
		wantStatus int
		wantName   string
		wantError  api.ErrorKey
	}{
		{
			name:       "not found",
			actor:      actor,
			code:       "XX",
			wantStatus: http.StatusNotFound,
			wantError:  api.ErrorResourceNotFound,
		},
		{
			name:       "found",
			actor:      actor,
			code:       "CH",
			wantStatus: http.StatusOK,
			wantName:   "Switzerland",
		},
	}
	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/%s/%s", domain.TypeCountry, tt.code)
			req := as.JSON(path)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Get()
			body := res.Body.Bytes()

			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned: %d\n%s", res.Code, body)
			if tt.wantStatus != http.StatusOK {
				var err api.AppError
				as.NoError(as.decodeBody(body, &err), "response data is not as expected")
				as.Equal(tt.wantError, err.Key, "error key is incorrect")
			} else {
				var country models.Country
				as.NoError(as.decodeBody(body, &country), "response data is not as expected")
				as.Equal(tt.code, country.Code, "Code is not as expected")
				as.Equal(tt.wantName, country.Name, "Name is not as expected")
			}
		})
	}
}

func (as *ActionSuite) Test_countriesList() {
	actor := models.CreateUserFixtures(as.DB, 1).Users[0]

	tests := []struct {
		name          string
		actor         models.User
		wantStatus    int
		wantCountries int
		wantError     api.ErrorKey
	}{
		{
			name:          "found",
			actor:         actor,
			wantStatus:    http.StatusOK,
			wantCountries: 242,
		},
	}
	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/" + domain.TypeCountry)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Get()
			body := res.Body.Bytes()

			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned: %d\n%s", res.Code, body)
			if tt.wantStatus != http.StatusOK {
				var err api.AppError
				as.NoError(as.decodeBody(body, &err), "response data is not as expected")
				as.Equal(tt.wantError, err.Key, "error key is incorrect")
			} else {
				var countries models.Countries
				as.NoError(as.decodeBody(body, &countries), "response data is not as expected")
				as.Equal(tt.wantCountries, len(countries), "wrong number of countries")
			}
		})
	}
}
