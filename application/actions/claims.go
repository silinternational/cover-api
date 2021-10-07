package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /claims Claims ClaimsList
//
// ClaimsList
//
// list all the current user's claims, or all Claims if called as an admin
//
// ---
// responses:
//   '200':
//     description: a list of Claims
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/Claim"
func claimsList(c buffalo.Context) error {
	user := models.CurrentUser(c)

	if user.IsAdmin() {
		return claimsListAll(c)
	}

	return claimsListMine(c)
}

func claimsListAll(c buffalo.Context) error {
	tx := models.Tx(c)
	var claims models.Claims
	if err := claims.All(tx); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, claims.ConvertToAPI(tx, models.User{}))
}

func claimsListMine(c buffalo.Context) error {
	tx := models.Tx(c)
	currentUser := models.CurrentUser(c)
	claims := currentUser.MyClaims(models.Tx(c))
	return renderOk(c, claims.ConvertToAPI(tx, currentUser))
}

// swagger:operation GET /claims/{id} Claims ClaimsView
//
// ClaimsView
//
// view a specific claim
//
// ---
// parameters:
// - name: id
//   in: path
//   required: true
//   description: claim ID
// responses:
//   '200':
//     description: a Claim
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsView(c buffalo.Context) error {
	tx := models.Tx(c)
	claim := getReferencedClaimFromCtx(c)
	return renderOk(c, claim.ConvertToAPI(tx, models.CurrentUser(c)))
}

// swagger:operation PUT /claims/{id} Claims ClaimsUpdate
//
// ClaimsUpdate
//
// update a claim
//
// ---
// parameters:
// - name: id
//   in: path
//   required: true
//   description: claim ID
// - name: claim input
//   in: body
//   description: claim create input object
//   required: true
//   schema:
//     "$ref": "#/definitions/ClaimUpdateInput"
// responses:
//   '200':
//     description: a Claim
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	claim := getReferencedClaimFromCtx(c)

	var input api.ClaimUpdateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	claim.IncidentType = input.IncidentType
	claim.IncidentDescription = input.IncidentDescription
	claim.IncidentDate = input.IncidentDate

	if err := claim.UpdateByUser(c); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, claim.ConvertToAPI(tx, models.CurrentUser(c)))
}

// swagger:operation POST /policies/{id}/claims Claims ClaimsCreate
//
// ClaimsCreate
//
// create a new Claim on a policy
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
//   - name: claim input
//     in: body
//     description: claim create input object
//     required: true
//     schema:
//       "$ref": "#/definitions/ClaimCreateInput"
// responses:
//   '200':
//     description: the new Claim
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsCreate(c buffalo.Context) error {
	policy := getReferencedPolicyFromCtx(c)

	var input api.ClaimCreateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)
	dbClaim, err := policy.AddClaim(tx, input)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, dbClaim.ConvertToAPI(tx, models.CurrentUser(c)))
}

// swagger:operation POST /claims/{id}/submit Claims ClaimsSubmit
//
// ClaimsSubmit
//
// Submit a claim for review.  Can be used at state "Draft" to submit for pre-approval or
//  "Receipt" to submit for payout approval.
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: claim ID
// responses:
//   '200':
//     description: submitted Claim
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsSubmit(c buffalo.Context) error {
	tx := models.Tx(c)
	claim := getReferencedClaimFromCtx(c)

	if err := claim.SubmitForApproval(c); err != nil {
		return reportError(c, err)
	}

	output := claim.ConvertToAPI(tx, models.CurrentUser(c))
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /claims/{id}/revision Claims ClaimsRequestRevision
//
// ClaimsRequestRevision
//
// Admin requests revisions on a claim.  Can be used at state "Review1" or "Review3".
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: claim ID
//   - name: claim revision input
//     in: body
//     description: claim request revision input object
//     required: true
//     schema:
//       "$ref": "#/definitions/ClaimStatusInput"
// responses:
//   '200':
//     description: Claim in focus
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsRequestRevision(c buffalo.Context) error {
	tx := models.Tx(c)
	claim := getReferencedClaimFromCtx(c)

	var input api.ClaimStatusInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	if err := claim.RequestRevision(c, input.StatusReason); err != nil {
		return reportError(c, err)
	}

	output := claim.ConvertToAPI(tx, models.CurrentUser(c))
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /claims/{id}/preapprove Claims ClaimsPreapprove
//
// ClaimsPreapprove
//
// Admin preapproves a claim and requests a receipt.  Can only be used at state "Review1".
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: claim ID
// responses:
//   '200':
//     description: Claim in focus
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsPreapprove(c buffalo.Context) error {
	tx := models.Tx(c)
	claim := getReferencedClaimFromCtx(c)

	if err := claim.RequestReceipt(c, ""); err != nil {
		return reportError(c, err)
	}

	output := claim.ConvertToAPI(tx, models.CurrentUser(c))
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /claims/{id}/receipt Claims ClaimsFixReceipt
//
// ClaimsFixReceipt
//
// Admin reverts a claim to request a new/better receipt.
// Can be used at state "Review2" or "Review3".
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: claim ID
//   - name: claim receipt reason input
//     in: body
//     description: claim receipt reason input object
//     required: true
//     schema:
//       "$ref": "#/definitions/ClaimStatusInput"
// responses:
//   '200':
//     description: Claim in focus
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsRequestReceipt(c buffalo.Context) error {
	tx := models.Tx(c)
	claim := getReferencedClaimFromCtx(c)

	var input api.ClaimStatusInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	if err := claim.RequestReceipt(c, input.StatusReason); err != nil {
		return reportError(c, err)
	}

	output := claim.ConvertToAPI(tx, models.CurrentUser(c))
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /claims/{id}/approve Claims ClaimsApprove
//
// ClaimsApprove
//
// Admin approves a claim.  Can be used at states "Review1","Review2","Review3".
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: claim ID
// responses:
//   '200':
//     description: Claim in focus
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsApprove(c buffalo.Context) error {
	tx := models.Tx(c)

	claim := getReferencedClaimFromCtx(c)

	if err := claim.Approve(c); err != nil {
		return reportError(c, err)
	}

	output := claim.ConvertToAPI(tx, models.CurrentUser(c))
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /claims/{id}/deny Claims ClaimsDeny
//
// ClaimsDeny
//
// Admin denies a claim.  Can be used at states "Review1","Review2","Review3".
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: claim ID
//   - name: claim deny input
//     in: body
//     description: claim deny input object
//     required: true
//     schema:
//       "$ref": "#/definitions/ClaimStatusInput"
// responses:
//   '200':
//     description: Claim in focus
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsDeny(c buffalo.Context) error {
	tx := models.Tx(c)

	claim := getReferencedClaimFromCtx(c)

	var input api.ClaimStatusInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	if err := claim.Deny(c, input.StatusReason); err != nil {
		return reportError(c, err)
	}

	output := claim.ConvertToAPI(tx, models.CurrentUser(c))
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /claims/{id}/items Claims ClaimsItemsCreate
//
// ClaimsItemsCreate
//
// create a new ClaimItem on a Claim
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: claim ID
//   - name: claim item input
//     in: body
//     description: claim item create input object
//     required: true
//     schema:
//       "$ref": "#/definitions/ClaimItemCreateInput"
// responses:
//   '200':
//     description: the new ClaimItem
//     schema:
//       "$ref": "#/definitions/ClaimItem"
func claimsItemsCreate(c buffalo.Context) error {
	claim := getReferencedClaimFromCtx(c)

	var input api.ClaimItemCreateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)
	claimItem, err := claim.AddItem(tx, input)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, claimItem.ConvertToAPI(tx))
}

// swagger:operation POST /claims/{id}/files Claims ClaimsFileAttach
//
// ClaimsFileAttach
//
// attach a File to a Claim
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: claim ID
//   - name: claim file input
//     in: body
//     description: claim file attach input object
//     required: true
//     schema:
//       "$ref": "#/definitions/ClaimFileAttachInput"
// responses:
//   '200':
//     description: the new ClaimFile
//     schema:
//       "$ref": "#/definitions/ClaimFile"
func claimsFilesAttach(c buffalo.Context) error {
	var input api.ClaimFileAttachInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)

	claim := getReferencedClaimFromCtx(c)
	claimFile, err := claim.AttachFile(tx, input)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, claimFile.ConvertToAPI(tx))
}

// getReferencedClaimFromCtx pulls the models.Claim resource from context that was put there
// by the AuthZ middleware
func getReferencedClaimFromCtx(c buffalo.Context) *models.Claim {
	claim, ok := c.Value(domain.TypeClaim).(*models.Claim)
	if !ok {
		panic("claim not found in context")
	}
	return claim
}
