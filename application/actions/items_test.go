package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/riskman-api/domain"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/models"
)

func (as *ActionSuite) Test_ItemsList() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	policies := fixtures.Policies
	item2 := fixtures.Items[2]
	item3 := fixtures.Items[3]

	normalUser := fixtures.Policies[1].Members[0]

	tests := []struct {
		name          string
		actor         models.User
		policy        models.Policy
		wantCount     int
		wantStatus    int
		wantInBody    []string
		notWantInBody string
	}{
		{
			name:          "unauthenticated",
			actor:         models.User{},
			wantStatus:    http.StatusUnauthorized,
			wantInBody:    []string{api.ErrorNotAuthorized.String()},
			notWantInBody: item2.ID.String(),
		},
		{
			name:          "uuid not found",
			actor:         normalUser,
			policy:        models.Policy{ID: domain.GetUUID()},
			wantStatus:    http.StatusNotFound,
			wantInBody:    []string{`"key":"` + api.ErrorResourceNotFound.String()},
			notWantInBody: item2.ID.String(),
		},
		{
			name:       "normal user good results",
			actor:      normalUser,
			policy:     policies[1],
			wantCount:  2,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`{"id":"` + item2.ID.String(),
				`"name":"` + item2.Name,
				`"category_id":"` + item2.CategoryID.String(),
				fmt.Sprintf(`"in_storage":%t`, item2.InStorage),
				`"country":"` + item2.Country,
				`"description":"` + item2.Description,
				`"make":"` + item2.Make,
				`"model":"` + item2.Model,
				`"serial_number":"` + item2.SerialNumber,
				fmt.Sprintf(`"coverage_amount":%v`, item2.CoverageAmount),
				`"coverage_status":"` + string(item2.CoverageStatus),
				`"coverage_start_date":"` + item2.CoverageStartDate.Format("2006-01-02"),
				`"category":{"id":"`,
				`"name":"` + item2.Name,
				//TODO add some checks for the Item Category
				`{"id":"` + item3.ID.String(),
			},
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies/%s/items", tt.policy.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "Items List")

			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}

			var items api.Items
			err := json.Unmarshal([]byte(body), &items)
			as.NoError(err)
			as.Equal(tt.wantCount, len(items), "incorrect count of items")
		})
	}
}
