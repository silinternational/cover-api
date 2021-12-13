package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_ClaimsList() {
	const numberOfPolicies = 3
	const claimsPerPolicy = 4
	const totalNumberOfClaims = claimsPerPolicy * numberOfPolicies
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    numberOfPolicies,
		UsersPerPolicy:      1,
		ClaimsPerPolicy:     claimsPerPolicy,
		ClaimItemsPerClaim:  2,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	appAdmin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]
	normalUser := fixtures.Policies[1].Members[0]

	fixtures.Claims[0].Status = api.ClaimStatusReview1
	as.NoError(as.DB.Update(&fixtures.Claims[0]))

	tests := []struct {
		name          string
		actor         models.User
		queryString   string
		wantStatus    int
		wantClaims    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "normal user",
			actor:         normalUser,
			wantStatus:    http.StatusOK,
			wantClaims:    claimsPerPolicy,
			wantInBody:    fixtures.Policies[1].Claims[0].ID.String(),
			notWantInBody: fixtures.Policies[0].Claims[0].ID.String(),
		},
		{
			name:       "admin user",
			actor:      appAdmin,
			wantStatus: http.StatusOK,
			wantClaims: 1,
			wantInBody: fixtures.Policies[0].Claims[0].ID.String(),
		},
		{
			name:        "admin user",
			actor:       appAdmin,
			queryString: "?status=" + string(api.ClaimStatusDraft),
			wantStatus:  http.StatusOK,
			wantClaims:  totalNumberOfClaims - 1,
			wantInBody:  fixtures.Policies[0].Claims[1].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/claims" + tt.queryString)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
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
			var responseObject api.Claims
			as.NoError(json.Unmarshal([]byte(body), &responseObject))
			as.Len(responseObject, tt.wantClaims, "incorrect # of claims, %+v", responseObject)
			for _, c := range responseObject {
				as.Len(c.Items, fixConfig.ItemsPerPolicy)
			}
		})
	}
}

func (as *ActionSuite) Test_PoliciesClaimsList() {
	const numberOfPolicies = 3
	const claimsPerPolicy = 4
	fixConfig := models.FixturesConfig{
		NumberOfPolicies: numberOfPolicies,
		ClaimsPerPolicy:  claimsPerPolicy,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	policy := fixtures.Policies[1]

	appAdmin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]
	normalUser := policy.Members[0]

	tests := []struct {
		name          string
		actor         models.User
		wantStatus    int
		wantClaims    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "normal user",
			actor:         normalUser,
			wantStatus:    http.StatusOK,
			wantClaims:    claimsPerPolicy,
			wantInBody:    policy.Claims[0].ID.String(),
			notWantInBody: fixtures.Policies[0].Claims[0].ID.String(),
		},
		{
			name:          "wrong user",
			actor:         fixtures.Policies[0].Members[0],
			wantStatus:    http.StatusNotFound,
			wantClaims:    claimsPerPolicy,
			notWantInBody: policy.Claims[0].ID.String(),
		},
		{
			name:       "admin user",
			actor:      appAdmin,
			wantStatus: http.StatusOK,
			wantClaims: claimsPerPolicy,
			wantInBody: policy.Claims[0].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/%s%s", policiesPath, policy.ID, claimsPath)
			req := as.JSON(url)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
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

			var responseObject api.Claims
			reader := strings.NewReader(body)
			decoder := json.NewDecoder(reader)
			decoder.DisallowUnknownFields()
			as.NoError(decoder.Decode(&responseObject))
			as.Len(responseObject, tt.wantClaims, "incorrect # of claims")
		})
	}
}

func (as *ActionSuite) Test_ClaimsView() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		UsersPerPolicy:      1,
		ClaimsPerPolicy:     4,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	// alias a couple users
	appAdmin := fixtures.Policies[0].Members[0]
	firstUser := fixtures.Policies[1].Members[0]
	secondUser := fixtures.Policies[2].Members[0]

	// make an admin
	appAdmin.AppRole = models.AppRoleSteward
	err := appAdmin.Update(as.DB)
	as.NoError(err, "failed to make an app admin")

	tests := []struct {
		name          string
		actor         models.User
		claim         models.Claim
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "unauthorized user",
			actor:         firstUser,
			claim:         fixtures.Policies[2].Claims[0],
			wantStatus:    http.StatusNotFound,
			notWantInBody: fixtures.Policies[2].ID.String(),
		},
		{
			name:       "authorized user",
			actor:      secondUser,
			claim:      fixtures.Policies[2].Claims[0],
			wantStatus: http.StatusOK,
			wantInBody: fixtures.Policies[2].Claims[0].ID.String(),
		},
		{
			name:       "admin user",
			actor:      appAdmin,
			claim:      fixtures.Policies[2].Claims[0],
			wantStatus: http.StatusOK,
			wantInBody: fixtures.Policies[2].Claims[0].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/claims/" + tt.claim.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				as.Contains(body, tt.wantInBody, "did not find expected string")
			}
			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody, "found unexpected string")
			}

			if res.Code != http.StatusOK {
				return
			}
			var responseObject api.Claim
			as.NoError(json.Unmarshal([]byte(body), &responseObject))
			as.Equal(tt.claim.ID, responseObject.ID, "incorrect object in response", responseObject)
		})
	}
}

func (as *ActionSuite) Test_ClaimsUpdate() {
	db := as.DB
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		UsersPerPolicy:      1,
		ClaimsPerPolicy:     4,
		ClaimItemsPerClaim:  1,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policy := fixtures.Policies[2]

	// alias a couple users
	appAdmin := fixtures.Policies[0].Members[0]
	firstUser := fixtures.Policies[1].Members[0]
	secondUser := policy.Members[0]

	// alias some claims
	draftClaim := policy.Claims[0]
	review1Claim := models.UpdateClaimStatus(db, policy.Claims[1], api.ClaimStatusReview1, "")
	approvedClaim := models.UpdateClaimStatus(db, policy.Claims[2], api.ClaimStatusApproved, "")

	// make an admin
	appAdmin.AppRole = models.AppRoleSteward
	err := appAdmin.Update(as.DB)
	as.NoError(err, "failed to make an app admin")

	input := api.ClaimUpdateInput{
		IncidentDate:        time.Now().UTC(),
		IncidentType:        api.ClaimIncidentTypeTheft,
		IncidentDescription: "a description",
	}

	tests := []struct {
		name          string
		actor         models.User
		claim         models.Claim
		input         api.ClaimUpdateInput
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "unauthorized user",
			actor:         firstUser,
			claim:         draftClaim,
			input:         input,
			wantStatus:    http.StatusNotFound,
			notWantInBody: policy.ID.String(),
		},
		{
			name:          "authorized user but bad status",
			actor:         secondUser,
			claim:         approvedClaim,
			input:         input,
			wantStatus:    http.StatusBadRequest,
			wantInBody:    string(api.ErrorClaimStatus),
			notWantInBody: policy.ID.String(),
		},
		{
			name:       "authorized user draft",
			actor:      secondUser,
			claim:      draftClaim,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: draftClaim.ID.String(),
		},
		{
			name:       "authorized user review1 to draft",
			actor:      secondUser,
			claim:      review1Claim,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: `"status":"` + string(api.ClaimStatusDraft),
		},
		{
			name:       "admin user",
			actor:      appAdmin,
			claim:      approvedClaim,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: `"status":"` + string(api.ClaimStatusApproved),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/claims/" + tt.claim.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Put(tt.input)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				as.Contains(body, tt.wantInBody, "did not find expected string")
			}
			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody, "found unexpected string")
			}

			if res.Code != http.StatusOK {
				return
			}
			var responseObject api.Claim
			as.NoError(json.Unmarshal([]byte(body), &responseObject))
			as.Equal(tt.claim.ID, responseObject.ID, "incorrect object in response", responseObject)

			updatedClaim := models.Claim{}
			as.NoError(as.DB.Find(&updatedClaim, tt.claim.ID))
			as.verifyClaimUpdate(input, updatedClaim)
		})
	}
}

func (as *ActionSuite) verifyClaimUpdate(input api.ClaimUpdateInput, claim models.Claim) {
	as.Equal(input.IncidentType, claim.IncidentType, "IncidentType not correct")
	as.Equal(input.IncidentDescription, claim.IncidentDescription, "IncidentDescription not correct")
	as.WithinDuration(input.IncidentDate, claim.IncidentDate, time.Millisecond, "IncidentDate not correct")
}

func (as *ActionSuite) Test_ClaimsCreate() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	policyByOther := fixtures.Policies[0]
	policyByUser := fixtures.Policies[1]
	policyByAdmin := fixtures.Policies[2]

	// alias a couple users
	appAdmin := fixtures.Policies[2].Members[0]
	normalUser := policyByUser.Members[0]

	// make an admin
	appAdmin.AppRole = models.AppRoleSteward
	err := appAdmin.Update(as.DB)
	as.NoError(err, "failed to make an app admin")

	input := api.ClaimCreateInput{
		IncidentDate:        time.Now(),
		IncidentType:        api.ClaimIncidentTypeTheft,
		IncidentDescription: "a description",
	}

	tests := []struct {
		name          string
		actor         models.User
		policy        models.Policy
		input         api.ClaimCreateInput
		wantStatus    int
		wantInBody    []string
		notWantInBody string
	}{
		{
			name:       "valid input",
			actor:      normalUser,
			policy:     policyByUser,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"policy_id":"` + policyByUser.ID.String(),
				`"incident_type":"` + string(input.IncidentType),
				`"incident_description":"` + input.IncidentDescription,
				`"status":"` + string(api.ClaimStatusDraft),
				`"claim_items":[]`,
			},
		},
		{
			name:          "other person's policy",
			actor:         normalUser,
			policy:        policyByOther,
			input:         input,
			wantStatus:    http.StatusNotFound,
			notWantInBody: policyByOther.ID.String(),
		},
		{
			name:       "admin operation on other person's policy",
			actor:      appAdmin,
			policy:     policyByAdmin,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"policy_id":"` + policyByAdmin.ID.String(),
				`"incident_type":"` + string(input.IncidentType),
				`"incident_description":"` + input.IncidentDescription,
				`"status":"` + string(api.ClaimStatusDraft),
				`"claim_items":[]`,
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(fmt.Sprintf("/policies/%s/claims", tt.policy.ID))
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(tt.input)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "Create Claim fields")

			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}
			var respObj api.Claim
			as.NoError(json.Unmarshal([]byte(body), &respObj))

			as.Equal(tt.input.IncidentDescription, respObj.IncidentDescription,
				"response object is not correct, %+v", respObj)
		})
	}
}

func (as *ActionSuite) Test_ClaimsItemsCreate() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
		ClaimsPerPolicy:     1,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	claim := fixtures.Policies[1].Claims[0]
	item := fixtures.Policies[1].Items[0]
	otherPolicyItem := fixtures.Policies[0].Items[0]

	otherUser := fixtures.Policies[0].Members[0]
	sameUser := fixtures.Policies[1].Members[0]

	isRepairable := true
	input := api.ClaimItemCreateInput{
		ItemID:          item.ID,
		IsRepairable:    &isRepairable,
		RepairEstimate:  200,
		RepairActual:    0,
		ReplaceEstimate: 300,
		ReplaceActual:   0,
		PayoutOption:    api.PayoutOptionRepair,
		FMV:             250,
	}

	inputItemIDNotFound := input
	inputItemIDNotFound.ItemID = domain.GetUUID()

	inputItemIDMismatch := input
	inputItemIDMismatch.ItemID = otherPolicyItem.ID

	tests := []struct {
		name          string
		actor         models.User
		claim         models.Claim
		input         api.ClaimItemCreateInput
		wantStatus    int
		wantInBody    []string
		notWantInBody string
	}{
		{
			name:       "item id not in database",
			actor:      sameUser,
			claim:      claim,
			input:      inputItemIDNotFound,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{string(api.ErrorResourceNotFound), "failed to load item"},
		},
		{
			name:       "item id from wrong policy",
			actor:      sameUser,
			claim:      claim,
			input:      inputItemIDMismatch,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{string(api.ErrorClaimItemCreateInvalidInput), "claim and item do not have same policy id"},
		},
		{
			name:       "valid input",
			actor:      sameUser,
			claim:      claim,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"item_id":"` + input.ItemID.String(),
				`"claim_id":"` + claim.ID.String(),
				`"status":"` + string(api.ClaimStatusDraft),
				fmt.Sprintf(`"is_repairable":%t`, *input.IsRepairable),
				fmt.Sprintf(`"repair_estimate":%v`, int(input.RepairEstimate)),
				fmt.Sprintf(`"replace_estimate":%v`, int(input.ReplaceEstimate)),
				`"payout_option":"` + string(input.PayoutOption),
				fmt.Sprintf(`"fmv":%v`, int(input.FMV)),
			},
		},
		{
			name:          "other person's policy",
			actor:         otherUser,
			claim:         claim,
			input:         input,
			wantStatus:    http.StatusNotFound,
			notWantInBody: claim.ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(fmt.Sprintf("/%s/%s/%s", domain.TypeClaim, tt.claim.ID, domain.TypeItem))
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"

			res := req.Post(tt.input)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "CreateItem Claim fields")

			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}
			var respObj api.ClaimItem
			as.NoError(json.Unmarshal([]byte(body), &respObj))

			as.Equal(tt.input.PayoutOption, respObj.PayoutOption,
				"response object is not correct, %+v", respObj)
		})
	}
}

func (as *ActionSuite) Test_ClaimsSubmit() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
		ClaimsPerPolicy:     3,
		ClaimItemsPerClaim:  1,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]

	draftClaim := policy.Claims[0]
	approvedClaim := models.UpdateClaimStatus(as.DB, policy.Claims[1], api.ClaimStatusApproved, "")

	goodParams := models.UpdateClaimItemsParams{
		PayoutOption: api.PayoutOptionFMV,
		FMV:          2000,
	}
	models.UpdateClaimItems(as.DB, draftClaim, goodParams)

	otherUser := fixtures.Policies[1].Members[0]

	tests := []struct {
		name       string
		actor      models.User
		oldClaim   models.Claim
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthorized user",
			actor:      otherUser,
			oldClaim:   draftClaim,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "bad start status",
			actor:      policyCreator,
			oldClaim:   approvedClaim,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{api.ErrorClaimStatus.String()},
		},
		{
			name:       "good claim",
			actor:      policyCreator,
			oldClaim:   draftClaim,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"incident_description":"` + draftClaim.IncidentDescription,
				`"status":"` + string(api.ClaimStatusReview1),
				`"status_change":"`,
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s", domain.TypeClaim, tt.oldClaim.ID.String(), api.ResourceSubmit)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(nil)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var claim models.Claim
			as.NoError(as.DB.Find(&claim, tt.oldClaim.ID),
				"error finding submitted item.")

			as.Equal(api.ClaimStatusReview1, claim.Status, "incorrect status after submission")
		})
	}
}

func (as *ActionSuite) Test_ClaimsRequestRevision() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
		ClaimsPerPolicy:     3,
		ClaimItemsPerClaim:  1,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]

	appAdmin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	draftClaim := policy.Claims[0]
	review1Claim := models.UpdateClaimStatus(as.DB, policy.Claims[1], api.ClaimStatusReview1, "")

	tests := []struct {
		name       string
		actor      models.User
		oldClaim   models.Claim
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "bad start status",
			actor:      appAdmin,
			oldClaim:   draftClaim,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{"invalid claim status transition from Draft to Revision"},
		},
		{
			name:       "non-admin user",
			actor:      policyCreator,
			oldClaim:   review1Claim,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "good claim",
			actor:      appAdmin,
			oldClaim:   review1Claim,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"incident_description":"` + review1Claim.IncidentDescription,
				`"status":"` + string(api.ClaimStatusRevision),
				`"status_change":"` + models.ClaimStatusChangeRevisions + appAdmin.Name(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s",
				domain.TypeClaim, tt.oldClaim.ID.String(), api.ResourceRevision)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			const message = "change all of it"
			res := req.Post(api.ClaimStatusInput{StatusReason: message})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var claim models.Claim
			as.NoError(as.DB.Find(&claim, tt.oldClaim.ID),
				"error finding submitted item.")

			as.Equal(api.ClaimStatusRevision, claim.Status, "incorrect status after submission")
			as.Equal(message, claim.StatusReason, "incorrect revision message")
		})
	}
}

func (as *ActionSuite) Test_ClaimsPreapprove() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
		ClaimsPerPolicy:     3,
		ClaimItemsPerClaim:  1,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]

	appAdmin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	draftClaim := policy.Claims[0]
	review1Claim := models.UpdateClaimStatus(as.DB, policy.Claims[1], api.ClaimStatusReview1, "")

	tests := []struct {
		name       string
		actor      models.User
		oldClaim   models.Claim
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "bad start status",
			actor:      appAdmin,
			oldClaim:   draftClaim,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{api.ErrorClaimStatus.String()},
		},
		{
			name:       "non-admin user",
			actor:      policyCreator,
			oldClaim:   review1Claim,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "good claim",
			actor:      appAdmin,
			oldClaim:   review1Claim,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"incident_description":"` + review1Claim.IncidentDescription,
				`"status":"` + string(api.ClaimStatusReceipt),
				`"status_change":"` + models.ClaimStatusChangeReceipt + appAdmin.Name(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s",
				domain.TypeClaim, tt.oldClaim.ID.String(), api.ResourcePreapprove)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(nil)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var claim models.Claim
			as.NoError(as.DB.Find(&claim, tt.oldClaim.ID),
				"error finding submitted item.")

			as.Equal(api.ClaimStatusReceipt, claim.Status, "incorrect status after submission")
		})
	}
}

func (as *ActionSuite) Test_ClaimsReceipt() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
		ClaimsPerPolicy:     3,
		ClaimItemsPerClaim:  1,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]

	appAdmin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	draftClaim := policy.Claims[0]
	review3Claim := models.UpdateClaimStatus(as.DB, policy.Claims[1], api.ClaimStatusReview3,
		"the final receipt, not just the quote")

	tests := []struct {
		name       string
		actor      models.User
		oldClaim   models.Claim
		reason     string
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "bad start status",
			actor:      appAdmin,
			oldClaim:   draftClaim,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{api.ErrorClaimStatus.String()},
		},
		{
			name:       "non-admin user",
			actor:      policyCreator,
			oldClaim:   review3Claim,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "good claim",
			actor:      appAdmin,
			oldClaim:   review3Claim,
			reason:     review3Claim.StatusReason,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"incident_description":"` + review3Claim.IncidentDescription,
				`"status_reason":"`, review3Claim.StatusReason,
				`"status":"` + string(api.ClaimStatusReceipt),
				`"status_change":"` + models.ClaimStatusChangeReceipt + appAdmin.Name(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s",
				domain.TypeClaim, tt.oldClaim.ID.String(), api.ResourceReceipt)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(api.ClaimStatusInput{StatusReason: tt.reason})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var claim models.Claim
			as.NoError(as.DB.Find(&claim, tt.oldClaim.ID),
				"error finding submitted item.")

			as.Equal(api.ClaimStatusReceipt, claim.Status, "incorrect status after submission")
			as.Equal(tt.reason, claim.StatusReason, "incorrect StatusReason after submission")
		})
	}
}

func (as *ActionSuite) Test_ClaimsApprove() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
		ClaimsPerPolicy:     4,
		ClaimItemsPerClaim:  1,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]

	steward := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]
	signator := models.CreateAdminUsers(as.DB)[models.AppRoleSignator]

	draftClaim := policy.Claims[0]

	// Make one of the claims requesting a FixedFraction payout
	ffClaim := models.UpdateClaimStatus(as.DB, policy.Claims[1], api.ClaimStatusReview1, "")
	ffClaim.IncidentType = api.ClaimIncidentTypeEvacuation
	as.NoError(as.DB.Update(&ffClaim), "error updating claim fixture")
	ffParams := models.UpdateClaimItemsParams{
		PayoutOption: api.PayoutOptionFixedFraction,
	}
	models.UpdateClaimItems(as.DB, ffClaim, ffParams)

	review2Claim := models.UpdateClaimStatus(as.DB, policy.Claims[2], api.ClaimStatusReview2, "")
	review3Claim := models.UpdateClaimStatus(as.DB, policy.Claims[3], api.ClaimStatusReview3, "")

	review3Claim.ReviewerID = nulls.NewUUID(steward.ID)
	as.NoError(as.DB.Update(&review3Claim), "error updating claim fixture")

	tests := []struct {
		name            string
		actor           models.User
		oldClaim        models.Claim
		wantStatus      int
		wantClaimStatus api.ClaimStatus
		wantInBody      []string
	}{
		{
			name:       "bad start status",
			actor:      steward,
			oldClaim:   draftClaim,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{api.ErrorClaimStatus.String()},
		},
		{
			name:       "non-admin user",
			actor:      policyCreator,
			oldClaim:   ffClaim,
			wantStatus: http.StatusNotFound,
		},
		{
			name:            "review1 to review3",
			actor:           steward,
			oldClaim:        ffClaim,
			wantStatus:      http.StatusOK,
			wantClaimStatus: api.ClaimStatusReview3,
			wantInBody: []string{
				`"incident_description":"` + ffClaim.IncidentDescription,
				`"status":"` + string(api.ClaimStatusReview3),
				`"status_change":"` + models.ClaimStatusChangeReview3 + steward.Name(),
				`"review_date":"` + time.Now().UTC().Format(domain.DateFormat),
				`"reviewer_id":"` + steward.ID.String(),
			},
		},
		{
			name:            "review2 to review3",
			actor:           steward,
			oldClaim:        review2Claim,
			wantStatus:      http.StatusOK,
			wantClaimStatus: api.ClaimStatusReview3,
			wantInBody: []string{
				`"incident_description":"` + review2Claim.IncidentDescription,
				`"status":"` + string(api.ClaimStatusReview3),
				`"status_change":"` + models.ClaimStatusChangeReview3 + steward.Name(),
				`"review_date":"` + time.Now().UTC().Format(domain.DateFormat),
				`"reviewer_id":"` + steward.ID.String(),
			},
		},
		{
			name:            "review3 to approved fail with same approver",
			actor:           steward,
			oldClaim:        review3Claim,
			wantStatus:      http.StatusBadRequest,
			wantClaimStatus: api.ClaimStatusApproved,
			wantInBody:      []string{api.ErrorClaimInvalidApprover.String()},
		},
		{
			name:            "review3 to approved OK with different approver",
			actor:           signator,
			oldClaim:        review3Claim,
			wantStatus:      http.StatusOK,
			wantClaimStatus: api.ClaimStatusApproved,
			wantInBody: []string{
				`"incident_description":"` + review3Claim.IncidentDescription,
				`"status":"` + string(api.ClaimStatusApproved),
				`"status_change":"` + models.ClaimStatusChangeApproved + signator.Name(),
				`"review_date":"` + time.Now().UTC().Format(domain.DateFormat),
				`"reviewer_id":"` + signator.ID.String(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s",
				domain.TypeClaim, tt.oldClaim.ID.String(), api.ResourceApprove)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(nil)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var claim models.Claim
			as.NoError(as.DB.Find(&claim, tt.oldClaim.ID),
				"error finding submitted item.")

			as.Equal(tt.wantClaimStatus, claim.Status, "incorrect status after submission")
		})
	}
}

func (as *ActionSuite) Test_ClaimsDeny() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
		ClaimsPerPolicy:     4,
		ClaimItemsPerClaim:  1,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policy := fixtures.Policies[0]
	policyCreator := policy.Members[0]

	steward := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]
	signator := models.CreateAdminUsers(as.DB)[models.AppRoleSignator]

	draftClaim := policy.Claims[0]
	review1Claim := models.UpdateClaimStatus(as.DB, policy.Claims[1], api.ClaimStatusReview1, "")
	review2Claim := models.UpdateClaimStatus(as.DB, policy.Claims[2], api.ClaimStatusReview2, "")
	review3Claim := models.UpdateClaimStatus(as.DB, policy.Claims[3], api.ClaimStatusReview3, "")

	tests := []struct {
		name            string
		actor           models.User
		oldClaim        models.Claim
		wantStatus      int
		wantClaimStatus api.ClaimStatus
		wantInBody      []string
	}{
		{
			name:       "bad start status",
			actor:      steward,
			oldClaim:   draftClaim,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{api.ErrorClaimStatus.String()},
		},
		{
			name:       "non-admin user",
			actor:      policyCreator,
			oldClaim:   review1Claim,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "review1 to denied",
			actor:      steward,
			oldClaim:   review1Claim,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"incident_description":"` + review1Claim.IncidentDescription,
				`"status":"` + string(api.ClaimStatusDenied),
				`"status_change":"` + models.ClaimStatusChangeDenied + steward.Name(),
				`"review_date":"` + time.Now().UTC().Format(domain.DateFormat),
				`"reviewer_id":"` + steward.ID.String(),
			},
		},
		{
			name:       "review2 to denied steward",
			actor:      steward,
			oldClaim:   review2Claim,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"incident_description":"` + review2Claim.IncidentDescription,
				`"status":"` + string(api.ClaimStatusDenied),
				`"status_change":"` + models.ClaimStatusChangeDenied + steward.Name(),
				`"review_date":"` + time.Now().UTC().Format(domain.DateFormat),
				`"reviewer_id":"` + steward.ID.String(),
			},
		},
		{
			name:       "review3 to denied signator",
			actor:      signator,
			oldClaim:   review3Claim,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"incident_description":"` + review3Claim.IncidentDescription,
				`"status":"` + string(api.ClaimStatusDenied),
				`"status_change":"` + models.ClaimStatusChangeDenied + signator.Name(),
				`"review_date":"` + time.Now().UTC().Format(domain.DateFormat),
				`"reviewer_id":"` + signator.ID.String(),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s/%s",
				domain.TypeClaim, tt.oldClaim.ID.String(), api.ResourceDeny)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			const message = "change all of it"
			res := req.Post(api.ClaimStatusInput{StatusReason: message})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var claim models.Claim
			as.NoError(as.DB.Find(&claim, tt.oldClaim.ID),
				"error finding submitted item.")

			as.Equal(api.ClaimStatusDenied, claim.Status, "incorrect status after submission")
			as.Equal(message, claim.StatusReason, "incorrect status reason")
		})
	}
}
