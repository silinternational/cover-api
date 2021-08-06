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
			policy:        models.Policy{ID: domain.GetUUID()},
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

func (as *ActionSuite) Test_ItemsAdd() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	policy := fixtures.Policies[0]
	policyCreator := fixtures.Policies[0].Members[0]
	otherUser := fixtures.Policies[1].Members[0]

	iCat := fixtures.ItemCategories[0]

	badItemDate := api.ItemInput{
		Name:       "Item with bad purchase date",
		CategoryID: domain.GetUUID(),
	}

	badCatID := api.ItemInput{
		Name:              "Item with missing category",
		CategoryID:        domain.GetUUID(),
		PurchaseDate:      "2006-01-02",
		CoverageStartDate: "2006-01-03",
		CoverageStatus:    api.ItemCoverageStatusDraft,
	}

	goodItem := api.ItemInput{
		Name:              "Good Item",
		CategoryID:        iCat.ID,
		InStorage:         true,
		Country:           "Thailand",
		Description:       "camera",
		Make:              "Minolta",
		Model:             "Max",
		SerialNumber:      "MM1234",
		CoverageAmount:    101,
		PurchaseDate:      "2006-01-02",
		CoverageStatus:    api.ItemCoverageStatusDraft,
		CoverageStartDate: "2006-01-03",
	}

	tests := []struct {
		name          string
		actor         models.User
		policy        models.Policy
		newItem       api.ItemInput
		wantStatus    int
		wantInBody    []string
		notWantInBody string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			policy:     policy,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:       "unauthorized",
			actor:      otherUser,
			policy:     policy,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{"actor not allowed to perform that action on this resource"},
		},
		{
			name:       "has bad purchase date",
			actor:      policyCreator,
			policy:     policy,
			newItem:    badItemDate,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{
				api.ErrorItemInvalidPurchaseDate.String(),
				"failed to parse item purchase date",
			},
		},
		{
			name:       "has bad category id",
			actor:      policyCreator,
			policy:     policy,
			newItem:    badCatID,
			wantStatus: http.StatusInternalServerError,
			wantInBody: []string{`violates foreign key constraint`},
		},
		{
			name:       "good item",
			actor:      policyCreator,
			policy:     policy,
			newItem:    goodItem,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"name":"` + goodItem.Name,
				`"category_id":"` + goodItem.CategoryID.String(),
				`"in_storage":true`,
				`"country":"` + goodItem.Country,
				`"description":"` + goodItem.Description,
				`"policy_id":"` + policy.ID.String(),
				`"make":"` + goodItem.Make,
				`"model":"` + goodItem.Model,
				`"serial_number":"` + goodItem.SerialNumber,
				fmt.Sprintf(`"coverage_amount":%v`, goodItem.CoverageAmount),
				`"purchase_date":"` + goodItem.PurchaseDate + `"`,
				`"coverage_status":"` + string(goodItem.CoverageStatus),
				`"category":{"id":"` + iCat.ID.String(),
				`"name":"` + iCat.Name,
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies/%s/items", tt.policy.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(tt.newItem)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "Items Add")

			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}

			var apiItem api.Item
			err := json.Unmarshal([]byte(body), &apiItem)
			as.NoError(err)

			var item models.Item
			as.NoError(as.DB.Where(`name = ?`, tt.newItem.Name).First(&item),
				"error finding newly added item.")
		})
	}
}

func (as *ActionSuite) Test_ItemsUpdate() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	oldItem := fixtures.Items[0]
	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]
	otherUser := fixtures.Policies[1].Members[0]

	iCat := fixtures.ItemCategories[1] // different one

	badItemDate := api.ItemInput{
		Name:       "Item with bad purchase date",
		CategoryID: oldItem.CategoryID,
	}

	badCatID := api.ItemInput{
		Name:              "Item with missing category",
		CategoryID:        domain.GetUUID(),
		PurchaseDate:      "2006-01-02",
		CoverageStartDate: "2006-01-03",
		CoverageStatus:    api.ItemCoverageStatusDraft,
	}

	goodItem := api.ItemInput{
		Name:              "Good Item",
		CategoryID:        iCat.ID,
		InStorage:         true,
		Country:           "Thailand",
		Description:       "camera",
		Make:              "Minolta",
		Model:             "Max",
		SerialNumber:      "MM1234",
		CoverageAmount:    oldItem.CoverageAmount,
		PurchaseDate:      "2006-01-02",
		CoverageStatus:    api.ItemCoverageStatusDraft,
		CoverageStartDate: "2006-01-03",
	}

	tests := []struct {
		name          string
		actor         models.User
		oldItem       models.Item
		newItem       api.ItemInput
		wantStatus    int
		wantInBody    []string
		notWantInBody string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			oldItem:    oldItem,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:       "unauthorized",
			actor:      otherUser,
			oldItem:    oldItem,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{"actor not allowed to perform that action on this resource"},
		},
		{
			name:       "bad item id",
			actor:      policyCreator,
			oldItem:    models.Item{ID: domain.GetUUID()},
			wantStatus: http.StatusNotFound,
			wantInBody: []string{api.ErrorResourceNotFound.String()},
		},
		{
			name:       "has bad purchase date",
			actor:      policyCreator,
			oldItem:    oldItem,
			newItem:    badItemDate,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{
				api.ErrorItemInvalidPurchaseDate.String(),
				"failed to parse item purchase date",
			},
		},
		{
			name:       "has bad category id",
			actor:      policyCreator,
			oldItem:    oldItem,
			newItem:    badCatID,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{api.ErrorQueryFailure.String()},
		},
		{
			name:       "good item",
			actor:      policyCreator,
			oldItem:    oldItem,
			newItem:    goodItem,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"name":"` + goodItem.Name,
				`"category_id":"` + goodItem.CategoryID.String(),
				`"in_storage":true`,
				`"country":"` + goodItem.Country,
				`"description":"` + goodItem.Description,
				`"policy_id":"` + policy.ID.String(),
				`"make":"` + goodItem.Make,
				`"model":"` + goodItem.Model,
				`"serial_number":"` + goodItem.SerialNumber,
				// keeps oldItem coverage_amount
				fmt.Sprintf(`"coverage_amount":%v`, goodItem.CoverageAmount),
				`"purchase_date":"` + goodItem.PurchaseDate + `"`,
				`"coverage_status":"` + string(goodItem.CoverageStatus),
				`"category":{"id":"` + iCat.ID.String(),
				`"name":"` + iCat.Name,
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/items/%s", tt.oldItem.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Put(tt.newItem)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "Items Add")

			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}

			var apiItem api.Item
			err := json.Unmarshal([]byte(body), &apiItem)
			as.NoError(err)

			var item models.Item
			as.NoError(as.DB.Where(`name = ?`, tt.newItem.Name).First(&item),
				"error finding newly added item.")
		})
	}
}
