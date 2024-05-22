package actions

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_ClaimFilesAttach() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:   2,
		ItemsPerPolicy:     2,
		UsersPerPolicy:     1,
		ClaimsPerPolicy:    4,
		ClaimItemsPerClaim: 1,
		ClaimFilesPerClaim: 1,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policyCreator := fixtures.Policies[0].Members[0]
	otherUser := fixtures.Policies[1].Members[0]
	claim := fixtures.Claims[0]
	newFileID := models.CreateFileFixtures(as.DB, 1, policyCreator.ID).Files[0].ID

	existingFileID := fixtures.Claims[0].ClaimFiles[0].FileID

	linkedFile := models.CreateFileFixtures(as.DB, 1, policyCreator.ID).Files[0]
	as.NoError(linkedFile.SetLinked(as.DB))

	tests := []struct {
		name       string
		actor      models.User
		claim      models.Claim
		request    api.ClaimFileAttachInput
		wantStatus int
		wantInBody string
	}{
		{
			name:       "not allowed",
			actor:      otherUser,
			claim:      claim,
			request:    api.ClaimFileAttachInput{FileID: newFileID},
			wantStatus: http.StatusNotFound,
			wantInBody: fmt.Sprintf(`"key":"%s"`, api.ErrorNotAuthorized),
		},
		{
			name:       "bad input",
			actor:      policyCreator,
			claim:      claim,
			request:    api.ClaimFileAttachInput{FileID: domain.GetUUID()},
			wantStatus: http.StatusBadRequest,
			wantInBody: fmt.Sprintf(`"key":"%s"`, api.ErrorForeignKeyViolation),
		},
		{
			name:       "file already linked to the claim",
			actor:      policyCreator,
			claim:      claim,
			request:    api.ClaimFileAttachInput{FileID: existingFileID},
			wantStatus: http.StatusBadRequest,
			wantInBody: fmt.Sprintf(`"key":"%s"`, api.ErrorUniqueKeyViolation),
		},
		{
			name:       "file linked to something else",
			actor:      policyCreator,
			claim:      claim,
			request:    api.ClaimFileAttachInput{FileID: linkedFile.ID},
			wantStatus: http.StatusBadRequest,
			wantInBody: fmt.Sprintf(`"key":"%s"`, api.ErrorFileAlreadyLinked),
		},
		{
			name:       "ok",
			actor:      policyCreator,
			claim:      claim,
			request:    api.ClaimFileAttachInput{FileID: newFileID},
			wantStatus: http.StatusOK,
			wantInBody: fmt.Sprintf(`"claim_id":"%s"`, claim.ID),
		},
	}
	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			claimID := tt.claim.ID
			req := as.JSON("/%s/%s/%s",
				domain.TypeClaim, claimID.String(), domain.TypeFile)
			req.Headers["Authorization"] = fmt.Sprintf("Access %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Post(tt.request)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData([]string{tt.wantInBody}, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var claimFile models.ClaimFile
			as.NoError(as.DB.Where("claim_id = ? AND file_id = ?", claimID, tt.request.FileID).First(&claimFile),
				"new ClaimFile not found in database")
		})
	}
}

func (as *ActionSuite) Test_ClaimFilesDelete() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:   2,
		ItemsPerPolicy:     2,
		ClaimsPerPolicy:    4,
		ClaimItemsPerClaim: 1,
		ClaimFilesPerClaim: 1,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policyCreator := fixtures.Policies[0].Members[0]
	otherUser := fixtures.Policies[1].Members[0]

	claimFileID := fixtures.Claims[0].ClaimFiles[0].ID

	var claimFiles models.ClaimFiles
	claimFileCount, err := as.DB.Count(&claimFiles)
	as.NoError(err, "error counting initial ClaimFiles")

	var files models.Files
	fileCount, err := as.DB.Count(&files)
	as.NoError(err, "error counting initial Files")

	tests := []struct {
		name       string
		actor      models.User
		id         uuid.UUID
		wantStatus int
		wantInBody string
	}{
		{
			name:       "not allowed",
			actor:      otherUser,
			id:         claimFileID,
			wantStatus: http.StatusNotFound,
			wantInBody: fmt.Sprintf(`"key":"%s"`, api.ErrorNotAuthorized),
		},
		{
			name:       "incorrect id",
			actor:      policyCreator,
			id:         domain.GetUUID(),
			wantStatus: http.StatusNotFound,
			wantInBody: fmt.Sprintf(`"key":"%s"`, api.ErrorResourceNotFound),
		},
		{
			name:       "ok",
			actor:      policyCreator,
			id:         claimFileID,
			wantStatus: http.StatusNoContent,
		},
	}
	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s", domain.TypeClaimFile, tt.id.String())
			req.Headers["Authorization"] = fmt.Sprintf("Access %s", tt.actor.Email)
			req.Headers["content-type"] = domain.ContentJson
			res := req.Delete()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			if res.Code != http.StatusNoContent {
				as.verifyResponseData([]string{tt.wantInBody}, body, "")
				return
			}

			var claimFiles models.ClaimFiles
			finalCount, err := as.DB.Count(&claimFiles)
			as.NoError(err, "error getting final count of ClaimFiles")
			as.Equal(claimFileCount-1, finalCount, "incorrect number of claim files left in db")

			var files models.Files
			finalFileCount, err := as.DB.Count(&files)
			as.NoError(err, "error getting final count of Files")
			as.Equal(fileCount-1, finalFileCount, "incorrect number of files left in db")
		})
	}
}
