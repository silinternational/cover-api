package actions

import (
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /claims Claims ClaimsList
//
// ClaimsList
//
// List the claims visible to the authenticated user, filtered by the given
// status values. For a user, all status values are included by default.
// For an admin (steward or signator) only the review status values are
// included by default. Accepted status values: Draft, Review1, Review2,
// Review3, Revision, Receipt, Approved, Paid, Denied
//
// ---
// parameters:
// - name: status
//   in: query
//   required: false
//   description: comma-separated list of status values to include
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
		statusParam := c.Param("status")
		var statusList []string
		if statusParam != "" {
			statusList = strings.Split(statusParam, ",")
		}
		statuses := make([]api.ClaimStatus, len(statusList))
		for i := range statusList {
			statuses[i] = api.ClaimStatus(statusList[i])
		}
		return claimsListAdmin(c, statuses)
	}

	return claimsListCustomer(c)
}

func claimsListAdmin(c buffalo.Context, statuses []api.ClaimStatus) error {
	tx := models.Tx(c)
	var claims models.Claims

	if err := claims.ByStatus(tx, statuses); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, claims.ConvertToAPI(tx))
}

func claimsListCustomer(c buffalo.Context) error {
	tx := models.Tx(c)
	currentUser := models.CurrentUser(c)
	claims := currentUser.MyClaims(models.Tx(c))
	return renderOk(c, claims.ConvertToAPI(tx))
}

// swagger:operation GET /policies/{id}/claims Claims PolicyClaimsList
//
// PolicyClaimsList
//
// List claims for a given policy
//
// ---
// responses:
//   '200':
//     description: a list of Claims
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/Claim"
func policiesClaimsList(c buffalo.Context) error {
	policy := getReferencedPolicyFromCtx(c)

	tx := models.Tx(c)

	policy.LoadClaims(tx, false)

	return renderOk(c, policy.Claims.ConvertToAPI(tx))
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
	return renderOk(c, claim.ConvertToAPI(tx))
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

	return renderOk(c, claim.ConvertToAPI(tx))
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
	dbClaim, err := policy.AddClaim(c, input)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, dbClaim.ConvertToAPI(tx))
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

	output := claim.ConvertToAPI(tx)
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /claims/{id}/revision Claims ClaimsRequestRevision
//
// ClaimsRequestRevision
//
// Admin requests revisions on a claim.  Can be used at state "Review1", "Review2", or "Review3".
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

	output := claim.ConvertToAPI(tx)
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

	output := claim.ConvertToAPI(tx)
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

	output := claim.ConvertToAPI(tx)
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

	output := claim.ConvertToAPI(tx)
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

	output := claim.ConvertToAPI(tx)
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
	claimItem, err := claim.AddItem(c, input)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, claimItem.ConvertToAPI(tx))
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
