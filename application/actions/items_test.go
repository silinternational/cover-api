package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
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

	item2.Load(as.DB)

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
				fmt.Sprintf(`"in_storage":%t`, item2.InStorage),
				`"country":"` + item2.Country,
				`"description":"` + item2.Description,
				`"policy_id":"` + item2.PolicyID.String(),
				`"make":"` + item2.Make,
				`"model":"` + item2.Model,
				`"serial_number":"` + item2.SerialNumber,
				fmt.Sprintf(`"coverage_amount":%v`, item2.CoverageAmount),
				`"coverage_status":"` + string(item2.CoverageStatus),
				`"coverage_start_date":"` + item2.CoverageStartDate.Format("2006-01-02"),
				`"name":"` + item2.Name,
				`"category":{"id":"` + item2.CategoryID.String(),
				`"name":"` + item2.Category.Name,
				`"risk_category":{"id":"` + item2.Category.RiskCategoryID.String(),
				`"name":"` + item2.Category.RiskCategory.Name,
				`{"id":"` + item3.ID.String(),
			},
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s", domain.TypePolicy, tt.policy.ID.String(), domain.TypeItem)
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

func (as *ActionSuite) Test_ItemsCreate() {
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

	riskCategoryMobileID := models.RiskCategoryMobileID()

	goodItem := api.ItemCreate{
		Name:                "Good Item",
		CategoryID:          iCat.ID,
		RiskCategoryID:      &riskCategoryMobileID,
		InStorage:           true,
		Country:             "Thailand",
		Description:         "camera",
		Make:                "Minolta",
		Model:               "Max",
		SerialNumber:        "MM1234",
		CoverageAmount:      101,
		CoverageStatus:      api.ItemCoverageStatusDraft,
		CoverageStartDate:   "2006-01-03",
		AccountablePersonID: policyCreator.ID,
	}

	badItemDate := goodItem
	badItemDate.CoverageStartDate = "1/1/2020"

	tests := []struct {
		name       string
		actor      models.User
		policy     models.Policy
		newItem    api.ItemCreate
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			policy:     policy,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{
				api.ErrorNotAuthorized.String(),
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
			name:       "bad request",
			actor:      policyCreator,
			policy:     policy,
			newItem:    badItemDate,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{api.ErrorItemInvalidCoverageStartDate.String()},
		},
		{
			name:       "ok",
			actor:      policyCreator,
			policy:     policy,
			newItem:    goodItem,
			wantStatus: http.StatusOK,
			wantInBody: []string{`"name":"` + goodItem.Name},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s", domain.TypePolicy, tt.policy.ID.String(), domain.TypeItem)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(tt.newItem)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "Items Create")

			if res.Code != http.StatusOK {
				return
			}

			var apiItem api.Item
			err := json.Unmarshal([]byte(body), &apiItem)
			as.NoError(err)

			var item models.Item
			as.NoError(as.DB.Where(`name = ?`, tt.newItem.Name).First(&item),
				"error finding newly created item.")
		})
	}
}

func (as *ActionSuite) Test_ItemsSubmit() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	approvedItem := models.UpdateItemStatus(as.DB, fixtures.Items[1], api.ItemCoverageStatusApproved, "")
	revisionItem := models.UpdateItemStatus(as.DB, fixtures.Items[0], api.ItemCoverageStatusRevision, "fix it")

	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]
	otherUser := fixtures.Policies[1].Members[0]

	iCatID := revisionItem.CategoryID

	tests := []struct {
		name       string
		actor      models.User
		oldItem    models.Item
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			oldItem:    revisionItem,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{
				api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:       "unauthorized",
			actor:      otherUser,
			oldItem:    revisionItem,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{"actor not allowed to perform that action on this resource"},
		},
		{
			name:       "bad start status",
			actor:      policyCreator,
			oldItem:    approvedItem,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "good item",
			actor:      policyCreator,
			oldItem:    revisionItem,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"name":"` + revisionItem.Name,
				fmt.Sprintf(`"in_storage":%t`, revisionItem.InStorage),
				`"country":"` + revisionItem.Country,
				`"description":"` + revisionItem.Description,
				`"policy_id":"` + policy.ID.String(),
				`"make":"` + revisionItem.Make,
				`"model":"` + revisionItem.Model,
				`"serial_number":"` + revisionItem.SerialNumber,
				// keeps revisionItem coverage_amount
				fmt.Sprintf(`"coverage_amount":%v`, revisionItem.CoverageAmount),
				`"coverage_start_date":"` + revisionItem.CoverageStartDate.Format(domain.DateFormat) + `"`,
				`"coverage_status":"` + string(api.ItemCoverageStatusApproved), // lower than auto-approve max
				`"category":{"id":"` + iCatID.String(),
				`"status_change":"` + models.ItemStatusChangeAutoApproved,
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s", domain.TypeItem, tt.oldItem.ID.String(), api.ResourceSubmit)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(nil)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var item models.Item
			as.NoError(as.DB.Find(&item, tt.oldItem.ID),
				"error finding submitted item.")

			as.Equal(api.ItemCoverageStatusApproved, item.CoverageStatus, "incorrect coverage status after submission")
		})
	}
}

func (as *ActionSuite) Test_ItemsRevision() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	approvedItem := models.UpdateItemStatus(as.DB, fixtures.Items[1], api.ItemCoverageStatusApproved, "")
	pendingItem := models.UpdateItemStatus(as.DB, fixtures.Items[0], api.ItemCoverageStatusPending, "")

	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]

	adminUser := fixtures.Policies[1].Members[0]
	adminUser.AppRole = models.AppRoleSteward
	as.NoError(as.DB.Save(&adminUser), "failed saving admin user")

	iCatID := pendingItem.CategoryID

	tests := []struct {
		name       string
		actor      models.User
		oldItem    models.Item
		reason     string
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			oldItem:    pendingItem,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{
				api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:       "owner unauthorized",
			actor:      policyCreator,
			oldItem:    pendingItem,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{"actor not allowed to perform that action on this resource"},
		},
		{
			name:       "bad start status",
			actor:      adminUser,
			oldItem:    approvedItem,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "missing status reason",
			actor:      adminUser,
			oldItem:    pendingItem,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{
				`"key":"` + string(api.ErrorValidation),
				`"message":"Item.StatusReason`,
			},
		},
		{
			name:       "good item",
			actor:      adminUser,
			oldItem:    pendingItem,
			reason:     "not up to snuff",
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"name":"` + pendingItem.Name,
				fmt.Sprintf(`"in_storage":%t`, pendingItem.InStorage),
				`"country":"` + pendingItem.Country,
				`"description":"` + pendingItem.Description,
				`"policy_id":"` + policy.ID.String(),
				`"make":"` + pendingItem.Make,
				`"model":"` + pendingItem.Model,
				`"serial_number":"` + pendingItem.SerialNumber,
				// keeps pendingItem coverage_amount
				fmt.Sprintf(`"coverage_amount":%v`, pendingItem.CoverageAmount),
				`"coverage_start_date":"` + pendingItem.CoverageStartDate.Format(domain.DateFormat) + `"`,
				`"coverage_status":"` + string(api.ItemCoverageStatusRevision),
				`"category":{"id":"` + iCatID.String(),
				`"status_change":"` + models.ItemStatusChangeRevisions + adminUser.Name(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s", domain.TypeItem, tt.oldItem.ID.String(), api.ResourceRevision)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(api.ItemStatusInput{StatusReason: tt.reason})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "Items Revision")

			if res.Code != http.StatusOK {
				return
			}

			var item models.Item
			as.NoError(as.DB.Find(&item, tt.oldItem.ID),
				"error finding submitted item.")

			as.Equal(api.ItemCoverageStatusRevision, item.CoverageStatus, "incorrect coverage status after submission")
		})
	}
}

func (as *ActionSuite) Test_ItemsApprove() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	approvedItem := models.UpdateItemStatus(as.DB, fixtures.Items[1], api.ItemCoverageStatusApproved, "")
	pendingItem := models.UpdateItemStatus(as.DB, fixtures.Items[0], api.ItemCoverageStatusPending, "")

	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]

	adminUser := fixtures.Policies[1].Members[0]
	adminUser.AppRole = models.AppRoleSteward
	as.NoError(as.DB.Save(&adminUser), "failed saving admin user")

	tests := []struct {
		name       string
		actor      models.User
		oldItem    models.Item
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			oldItem:    pendingItem,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{
				api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:       "owner unauthorized",
			actor:      policyCreator,
			oldItem:    pendingItem,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{"actor not allowed to perform that action on this resource"},
		},
		{
			name:       "bad start status",
			actor:      adminUser,
			oldItem:    approvedItem,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "good item",
			actor:      adminUser,
			oldItem:    pendingItem,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"name":"` + pendingItem.Name,
				// other fields are tested in the revision test above
				`"coverage_status":"` + string(api.ItemCoverageStatusApproved),
				`"status_change":"` + models.ItemStatusChangeApproved + adminUser.Name(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s", domain.TypeItem, tt.oldItem.ID.String(), api.ResourceApprove)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(nil)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "Items Approve")

			if res.Code != http.StatusOK {
				return
			}

			var item models.Item
			as.NoError(as.DB.Find(&item, tt.oldItem.ID),
				"error finding submitted item.")

			as.Equal(api.ItemCoverageStatusApproved, item.CoverageStatus, "incorrect coverage status after submission")
		})
	}
}

func (as *ActionSuite) Test_ItemsDeny() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	approvedItem := models.UpdateItemStatus(as.DB, fixtures.Items[1], api.ItemCoverageStatusApproved, "")
	pendingItem := models.UpdateItemStatus(as.DB, fixtures.Items[0], api.ItemCoverageStatusPending, "")

	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]

	adminUser := fixtures.Policies[1].Members[0]
	adminUser.AppRole = models.AppRoleSteward
	as.NoError(as.DB.Save(&adminUser), "failed saving admin user")

	tests := []struct {
		name       string
		actor      models.User
		oldItem    models.Item
		reason     string
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			oldItem:    pendingItem,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{
				api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:       "owner unauthorized",
			actor:      policyCreator,
			oldItem:    pendingItem,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{"actor not allowed to perform that action on this resource"},
		},
		{
			name:       "bad start status",
			actor:      adminUser,
			oldItem:    approvedItem,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "missing reason",
			actor:      adminUser,
			oldItem:    pendingItem,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{
				`"key":"` + string(api.ErrorValidation),
				`"message":"Item.StatusReason`,
			},
		},
		{
			name:       "good item",
			actor:      adminUser,
			oldItem:    pendingItem,
			reason:     "spacecraft are not covered",
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"name":"` + pendingItem.Name,
				// other fields are tested in the revision test above
				`"coverage_status":"` + string(api.ItemCoverageStatusDenied),
				`"status_change":"` + models.ItemStatusChangeDenied + adminUser.Name(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s", domain.TypeItem, tt.oldItem.ID.String(), api.ResourceDeny)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"

			res := req.Post(api.ItemStatusInput{StatusReason: tt.reason})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "Items Deny")

			if res.Code != http.StatusOK {
				return
			}

			var item models.Item
			as.NoError(as.DB.Find(&item, tt.oldItem.ID),
				"error finding submitted item.")

			as.Equal(api.ItemCoverageStatusDenied, item.CoverageStatus, "incorrect coverage status after submission")
			as.Equal(tt.reason, item.StatusReason, "incorrect coverage status after submission")
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

	revisionItem := models.UpdateItemStatus(as.DB, fixtures.Items[0], api.ItemCoverageStatusRevision, "too many tribbles")
	approvedItem := models.UpdateItemStatus(as.DB, fixtures.Items[1], api.ItemCoverageStatusApproved, "")

	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]
	otherUser := fixtures.Policies[1].Members[0]

	iCat := fixtures.ItemCategories[1] // different one

	badCatID := api.ItemUpdate{
		Name:       "Item with missing category",
		CategoryID: domain.GetUUID(),
	}

	riskCategoryMobileID := models.RiskCategoryMobileID()

	goodItem := api.ItemUpdate{
		Name:                "Good Item",
		CategoryID:          iCat.ID,
		RiskCategoryID:      &riskCategoryMobileID,
		InStorage:           true,
		Country:             "Thailand",
		Description:         "camera",
		Make:                "Minolta",
		Model:               "Max",
		SerialNumber:        "MM1234",
		CoverageStatus:      api.ItemCoverageStatusRevision,
		AccountablePersonID: policyCreator.ID,
	}

	tests := []struct {
		name       string
		actor      models.User
		oldItem    models.Item
		newItem    api.ItemUpdate
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			oldItem:    revisionItem,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{
				api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:       "unauthorized",
			actor:      otherUser,
			oldItem:    revisionItem,
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
			name:       "has bad category id",
			actor:      policyCreator,
			oldItem:    revisionItem,
			newItem:    badCatID,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{api.ErrorInvalidCategory.String()},
		},
		{
			name:       "has bad start status",
			actor:      policyCreator,
			oldItem:    approvedItem,
			newItem:    badCatID,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "good item",
			actor:      policyCreator,
			oldItem:    revisionItem,
			newItem:    goodItem,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"name":"` + goodItem.Name,
				`"in_storage":true`,
				`"country":"` + goodItem.Country,
				`"description":"` + goodItem.Description,
				`"policy_id":"` + policy.ID.String(),
				`"make":"` + goodItem.Make,
				`"model":"` + goodItem.Model,
				`"serial_number":"` + goodItem.SerialNumber,
				// keeps oldItem coverage_amount
				fmt.Sprintf(`"coverage_amount":%v`, revisionItem.CoverageAmount),
				`"coverage_start_date":"` + revisionItem.CoverageStartDate.Format(domain.DateFormat) + `"`,
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

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var apiItem api.Item
			err := json.Unmarshal([]byte(body), &apiItem)
			as.NoError(err)

			var item models.Item
			as.NoError(as.DB.Where(`name = ?`, tt.newItem.Name).First(&item),
				"error finding newly updated item.")
		})
	}
}

func (as *ActionSuite) Test_ItemsRemove() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	item2 := fixtures.Items[2]
	item3 := fixtures.Items[3]

	oldHours := time.Duration(time.Hour * -(domain.ItemDeleteCutOffHours + 4))
	oldTime := time.Now().UTC().Add(oldHours)

	oldItem := fixtures.Items[4]
	oldItem.CreatedAt = oldTime

	q := "UPDATE items SET created_at = ? WHERE id = ?"
	err := as.DB.RawQuery(q, oldTime.Format(time.RFC3339), oldItem.ID.String()).Exec()
	as.NoError(err, "error updating item to look old")

	adminUser := fixtures.Policies[0].Members[0]
	adminUser.AppRole = models.AppRoleSteward
	as.NoError(as.DB.Save(&adminUser), "failed saving admin user")

	policyOwner := fixtures.Policies[1].Members[0]
	otherUser := fixtures.Policies[2].Members[0]

	tests := []struct {
		name           string
		actor          models.User
		item           models.Item
		wantCount      int
		wantHTTPStatus int
		wantItemStatus api.ItemCoverageStatus
		wantInBody     []string
	}{
		{
			name:           "unauthenticated",
			actor:          models.User{},
			item:           item2,
			wantCount:      4,
			wantHTTPStatus: http.StatusUnauthorized,
			wantInBody: []string{
				api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:           "unauthorized",
			actor:          otherUser,
			item:           item2,
			wantCount:      4,
			wantHTTPStatus: http.StatusNotFound,
			wantInBody:     []string{"actor not allowed to perform that action on this resource"},
		},
		{
			name:           "bad item id",
			actor:          policyOwner,
			item:           models.Item{ID: domain.GetUUID()},
			wantHTTPStatus: http.StatusNotFound,
		},
		{
			name:           "inactivate old item",
			actor:          adminUser,
			item:           oldItem,
			wantCount:      6,
			wantHTTPStatus: http.StatusNoContent,
			wantItemStatus: api.ItemCoverageStatusInactive,
		},
		{
			name:           "ok for policy creator",
			actor:          policyOwner,
			item:           item2,
			wantCount:      5,
			wantHTTPStatus: http.StatusNoContent,
		},
		{
			name:           "ok for admin",
			actor:          adminUser,
			item:           item3,
			wantCount:      4,
			wantHTTPStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s", domain.TypeItem, tt.item.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Delete()

			body := res.Body.String()
			as.Equal(tt.wantHTTPStatus, res.Code, "incorrect status code returned, body: %s", body)

			if res.Code != http.StatusNoContent {
				as.verifyResponseData(tt.wantInBody, body, "")
				return
			}

			var dbItems models.Items
			count, err := as.DB.Count(&dbItems)
			as.NoError(err)

			as.Equal(tt.wantCount, count, "incorrect number of remaining items")

			if string(tt.wantItemStatus) != "" {
				dbItem := models.Item{}
				as.NoError(as.DB.Find(&dbItem, tt.item.ID))
				as.Equal(tt.wantItemStatus, dbItem.CoverageStatus, "incorrect item status")
			}
		})
	}
}

func (as *ActionSuite) Test_NewItemFromApiInput() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	user := fixtures.Users[0]
	admin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	policy := fixtures.Policies[0]

	itemCategory := fixtures.ItemCategories[0]

	item := api.ItemCreate{
		Name:                "Good Item",
		CategoryID:          itemCategory.ID,
		InStorage:           true,
		Country:             "Thailand",
		Description:         "camera",
		Make:                "Minolta",
		Model:               "Max",
		SerialNumber:        "MM1234",
		CoverageAmount:      101,
		CoverageStatus:      api.ItemCoverageStatusDraft,
		CoverageStartDate:   "2006-01-03",
		AccountablePersonID: user.ID,
	}

	itemWithBadCoverageStartDate := item
	itemWithBadCoverageStartDate.Name = "Item with bad coverage start date"
	itemWithBadCoverageStartDate.CoverageStartDate = "1/1/2020"

	itemWithBadCategory := item
	itemWithBadCategory.Name = "Item with bad category"
	itemWithBadCategory.CategoryID = domain.GetUUID()

	itemWithNoRiskCategory := item
	itemWithNoRiskCategory.Name = "Item with no risk category"

	itemWithRiskCategory := item
	itemWithRiskCategory.Name = "Item with a specified risk category"
	riskCategoryMobileID := models.RiskCategoryMobileID()
	itemWithRiskCategory.RiskCategoryID = &riskCategoryMobileID

	tests := []struct {
		name        string
		policy      models.Policy
		input       api.ItemCreate
		user        models.User
		wantErr     string
		wantErrKey  api.ErrorKey
		wantErrCat  api.ErrorCategory
		wantRiskCat uuid.UUID
	}{
		{
			name:       itemWithBadCoverageStartDate.Name,
			policy:     policy,
			input:      itemWithBadCoverageStartDate,
			user:       user,
			wantErr:    "failed to parse item coverage start date",
			wantErrKey: api.ErrorItemInvalidCoverageStartDate,
			wantErrCat: api.CategoryUser,
		},
		{
			name:       itemWithBadCategory.Name,
			policy:     policy,
			input:      itemWithBadCategory,
			user:       user,
			wantErr:    "invalid category",
			wantErrKey: api.ErrorInvalidCategory,
			wantErrCat: api.CategoryUser,
		},
		{
			name:        itemWithNoRiskCategory.Name,
			policy:      policy,
			input:       itemWithRiskCategory,
			user:        user,
			wantRiskCat: itemCategory.RiskCategoryID,
		},
		{
			name:        itemWithRiskCategory.Name + " normal user",
			policy:      policy,
			input:       itemWithRiskCategory,
			user:        user,
			wantRiskCat: itemCategory.RiskCategoryID, // normal user cannot override
		},
		{
			name:        itemWithRiskCategory.Name + " admin user",
			policy:      policy,
			input:       itemWithRiskCategory,
			user:        admin,
			wantRiskCat: *itemWithRiskCategory.RiskCategoryID, // admin user can override
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			got, err := models.NewItemFromApiInput(models.CreateTestContext(tt.user), tt.input, tt.policy.ID)

			if tt.wantErr != "" {
				as.Error(err, "UUT did not return expected error")
				var appErr *api.AppError
				as.True(errors.As(err, &appErr), "UUT returned an error that is not an AppError")
				as.Contains(appErr.Error(), tt.wantErr, "error message is not correct")
				as.Equal(appErr.Key, tt.wantErrKey, "error key is not correct")
				as.Equal(appErr.Category, tt.wantErrCat, "error category is not correct")
				return
			}

			as.NoError(err, "UUT returned an unexpected error")

			as.Equal(tt.wantRiskCat, got.RiskCategoryID, "RiskCategoryID is not correct")
			as.Equal(tt.policy.ID, got.PolicyID, "PolicyID is not correct")

			as.Equal(tt.input.Name, got.Name, "Name is not correct")
			as.Equal(tt.input.CategoryID, got.CategoryID, "CategoryID is not correct")
			as.Equal(tt.input.InStorage, got.InStorage, "InStorage is not correct")
			as.Equal(tt.input.Country, got.Country, "Country is not correct")
			as.Equal(tt.input.Description, got.Description, "Description is not correct")
			as.Equal(tt.input.Make, got.Make, "Make is not correct")
			as.Equal(tt.input.Model, got.Model, "Model is not correct")
			as.Equal(tt.input.SerialNumber, got.SerialNumber, "SerialNumber is not correct")
			as.Equal(tt.input.CoverageAmount, got.CoverageAmount, "CoverageAmount is not correct")
			as.Equal(tt.input.CoverageStatus, got.CoverageStatus, "CoverageStatus is not correct")
			as.Equal("", got.StatusChange, "StatusChange is not correct")
		})
	}
}
