package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /policies/{id}/items PolicyItems PolicyItemsList
//
// PolicyItemsList
//
// gets the data for all the items on a Policy
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
// responses:
//   '200':
//     description: all policy items
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/Item"
func itemsList(c buffalo.Context) error {
	tx := models.Tx(c)

	policy := getReferencedPolicyFromCtx(c)

	policy.LoadItems(tx, true)

	return renderOk(c, policy.Items.ConvertToAPI(tx))
}

// swagger:operation POST /policies/{id}/items PolicyItems PolicyItemsCreate
//
// PolicyItemsCreate
//
// create a policy item
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
//   - name: policy item create input
//     in: body
//     description: policy item create input object
//     required: true
//     schema:
//       "$ref": "#/definitions/ItemInput"
// responses:
//   '200':
//     description: new Item
//     schema:
//       "$ref": "#/definitions/Item"
func itemsCreate(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)

	var itemPost api.ItemInput
	if err := StrictBind(c, &itemPost); err != nil {
		return reportError(c, err)
	}

	item, err := models.NewItemFromApiInput(c, itemPost, policy.ID)
	if err != nil {
		return reportError(c, err)
	}

	if err := item.Create(tx); err != nil {
		return reportError(c, err)
	}

	output := item.ConvertToAPI(tx)

	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation PUT /items/{id} PolicyItems PolicyItemsUpdate
//
// PolicyItemsUpdate
//
// update a policy item
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: item ID
//   - name: policy item update input
//     in: body
//     description: policy item create update object
//     required: true
//     schema:
//       "$ref": "#/definitions/ItemInput"
// responses:
//   '200':
//     description: updated Item
//     schema:
//       "$ref": "#/definitions/Item"
func itemsUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	item := getReferencedItemFromCtx(c)

	var itemPut api.ItemInput
	if err := StrictBind(c, &itemPut); err != nil {
		return reportError(c, err)
	}

	if item.CategoryID != itemPut.CategoryID {
		var iCat models.ItemCategory
		if err := iCat.FindByID(tx, itemPut.CategoryID); err != nil {
			return reportError(c, err)
		}
	}

	newItem, err := models.NewItemFromApiInput(c, itemPut, item.PolicyID)
	if err != nil {
		return reportError(c, err)
	}
	newItem.ID = item.ID
	newItem.StatusReason = item.StatusReason // don't let this change through an update

	if err := newItem.Update(c); err != nil {
		return reportError(c, err)
	}

	output := newItem.ConvertToAPI(tx)
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /items/{id}/submit PolicyItems PolicyItemsSubmit
//
// PolicyItemsSubmit
//
// submit a policy item for coverage
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: item ID
// responses:
//   '200':
//     description: submitted Item
//     schema:
//       "$ref": "#/definitions/Item"
func itemsSubmit(c buffalo.Context) error {
	tx := models.Tx(c)
	item := getReferencedItemFromCtx(c)

	if err := item.SubmitForApproval(c); err != nil {
		return reportError(c, err)
	}

	output := item.ConvertToAPI(tx)
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /items/{id}/revision PolicyItems PolicyItemsRevision
//
// PolicyItemsRevision
//
// admin requires changes on a policy item
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: item ID
//   - name: item revision input
//     in: body
//     description: item revision input object
//     required: true
//     schema:
//       "$ref": "#/definitions/ItemStatusInput"
// responses:
//   '200':
//     description: Policy Item
//     schema:
//       "$ref": "#/definitions/Item"
func itemsRevision(c buffalo.Context) error {
	tx := models.Tx(c)
	item := getReferencedItemFromCtx(c)

	var input api.ItemStatusInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	if err := item.Revision(c, input.StatusReason); err != nil {
		return reportError(c, err)
	}

	output := item.ConvertToAPI(tx)
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /items/{id}/approve PolicyItems PolicyItemsApprove
//
// PolicyItemsApprove
//
// approve coverage on a policy item
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: item ID
// responses:
//   '200':
//     description: approved Item
//     schema:
//       "$ref": "#/definitions/Item"
func itemsApprove(c buffalo.Context) error {
	tx := models.Tx(c)
	item := getReferencedItemFromCtx(c)

	if err := item.Approve(c, true); err != nil {
		return reportError(c, err)
	}

	output := item.ConvertToAPI(tx)
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation POST /items/{id}/deny PolicyItems PolicyItemsDeny
//
// PolicyItemsDeny
//
// deny coverage on a policy item
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: item ID
//   - name: item denial input
//     in: body
//     description: item denial input object
//     required: true
//     schema:
//       "$ref": "#/definitions/ItemStatusInput"
// responses:
//   '200':
//     description: denied Item
//     schema:
//       "$ref": "#/definitions/Item"
func itemsDeny(c buffalo.Context) error {
	tx := models.Tx(c)
	item := getReferencedItemFromCtx(c)

	var input api.ItemStatusInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	if err := item.Deny(c, input.StatusReason); err != nil {
		return reportError(c, err)
	}

	output := item.ConvertToAPI(tx)
	return c.Render(http.StatusOK, r.JSON(output))
}

// swagger:operation DELETE /items/{id} PolicyItems PolicyItemsRemove
//
// PolicyItemsRemove
//
// Delete a policy item if it is less than 72 hours old and has no associations.
//   Otherwise, inactivate coverage on it.
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: item ID
// responses:
//   '204':
//     description: OK but no content in response
func itemsRemove(c buffalo.Context) error {
	item := getReferencedItemFromCtx(c)

	if err := item.SafeDeleteOrInactivate(c); err != nil {
		return reportError(c, err)
	}

	return c.Render(http.StatusNoContent, nil)
}

// getReferencedItemFromCtx pulls the models.Item resource from context that was put there
// by the AuthZ middleware
func getReferencedItemFromCtx(c buffalo.Context) *models.Item {
	item, ok := c.Value(domain.TypeItem).(*models.Item)
	if !ok {
		panic("item not found in context")
	}
	return item
}
