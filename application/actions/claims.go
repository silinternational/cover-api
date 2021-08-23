package actions

import (
	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
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
	if err := models.Tx(c).All(&claims); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, models.ConvertClaims(tx, claims))
}

func claimsListMine(c buffalo.Context) error {
	tx := models.Tx(c)
	currentUser := models.CurrentUser(c)
	claims := currentUser.MyClaims(models.Tx(c))
	return renderOk(c, models.ConvertClaims(tx, claims))
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
	if claim == nil {
		panic("claim not found in context")
	}
	return renderOk(c, models.ConvertClaim(tx, *claim))
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
// responses:
//   '200':
//     description: a Claim
//     schema:
//       "$ref": "#/definitions/Claim"
func claimsUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	claim := getReferencedClaimFromCtx(c)
	if claim == nil {
		panic("claim not found in context")
	}

	var input api.ClaimUpdateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorClaimUpdateInvalidInput, api.CategoryUser))
	}

	// for future proofing
	oldStatus := claim.Status

	claim.EventType = input.EventType
	claim.EventDescription = input.EventDescription
	claim.EventDate = input.EventDate

	if err := claim.Update(models.Tx(c), oldStatus); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, models.ConvertClaim(tx, *claim))
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
	if policy == nil {
		panic("policy not found in route")
	}

	var input api.ClaimCreateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorClaimCreateInvalidInput, api.CategoryUser))
	}

	tx := models.Tx(c)
	dbClaim, err := policy.AddClaim(tx, input)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, models.ConvertClaim(tx, dbClaim))
}

// swagger:operation POST /policies/{id}/claims Claims ClaimsItemsCreate
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
//     description: policy ID
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
	if claim == nil {
		panic("claim not found in route")
	}

	var input api.ClaimItemCreateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorClaimItemCreateInvalidInput, api.CategoryUser))
	}

	tx := models.Tx(c)
	claimItem, err := claim.AddItem(tx, *claim, input)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, models.ConvertClaimItem(tx, claimItem))
}

// getReferencedClaimFromCtx pulls the models.Claim resource from context that was put there
// by the AuthZ middleware
func getReferencedClaimFromCtx(c buffalo.Context) *models.Claim {
	claim, ok := c.Value(domain.TypeClaim).(*models.Claim)
	if !ok {
		return nil
	}
	return claim
}
