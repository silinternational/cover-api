package actions

import (
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation PUT /claim-items/{id} ClaimItems ClaimItemsUpdate
// ClaimItemsUpdate
//
// update a claim item
// ---
//
//	parameters:
//	- name: id
//	  in: path
//	  required: true
//	  description: claim item ID
//	- name: claim item input
//	  in: body
//	  description: claim item update input object
//	  required: true
//	  schema:
//	    "$ref": "#/definitions/ClaimItemUpdateInput"
//	responses:
//	  '200':
//	    description: the updated ClaimItem
//	    schema:
//	      "$ref": "#/definitions/ClaimItem"
func claimItemsUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	claimItem := getReferencedClaimItemFromCtx(c)

	var input api.ClaimItemUpdateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	if input.IsRepairable != nil {
		claimItem.IsRepairable = nulls.NewBool(*input.IsRepairable)
	}
	claimItem.RepairEstimate = input.RepairEstimate
	claimItem.RepairActual = input.RepairActual
	claimItem.ReplaceEstimate = input.ReplaceEstimate
	claimItem.ReplaceActual = input.ReplaceActual
	claimItem.PayoutOption = input.PayoutOption
	claimItem.FMV = input.FMV

	if err := claimItem.Update(c); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, claimItem.ConvertToAPI(tx))
}

// getReferencedClaimItemFromCtx pulls the models.ClaimItem resource from context that was put there
// by the AuthZ middleware
func getReferencedClaimItemFromCtx(c buffalo.Context) *models.ClaimItem {
	claimItem, ok := c.Value(domain.TypeClaimItem).(*models.ClaimItem)
	if !ok {
		panic("claim item not found in context")
	}
	return claimItem
}
