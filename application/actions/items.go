package actions

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

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

	return renderOk(c, models.ConvertItems(tx, policy.Items))
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

	item, err := convertItemApiInput(c, itemPost, policy.ID)
	if err != nil {
		return reportError(c, err)
	}

	if err := item.Create(tx); err != nil {
		return reportError(c, err)
	}

	output := models.ConvertItem(tx, item)

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

	newItem, err := convertItemApiInput(c, itemPut, item.PolicyID)
	if err != nil {
		return reportError(c, err)
	}
	newItem.ID = item.ID

	if err := newItem.Update(tx, item.CoverageStatus); err != nil {
		return reportError(c, err)
	}

	output := models.ConvertItem(tx, newItem)
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

	if err := item.SubmitForApproval(tx); err != nil {
		return reportError(c, err)
	}

	output := models.ConvertItem(tx, *item)
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

	if err := item.Approve(tx); err != nil {
		return reportError(c, err)
	}

	output := models.ConvertItem(tx, *item)
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
// responses:
//   '200':
//     description: denied Item
//     schema:
//       "$ref": "#/definitions/Item"
func itemsDeny(c buffalo.Context) error {
	tx := models.Tx(c)
	item := getReferencedItemFromCtx(c)

	if err := item.Deny(tx); err != nil {
		return reportError(c, err)
	}

	output := models.ConvertItem(tx, *item)
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
	tx := models.Tx(c)
	item := getReferencedItemFromCtx(c)

	user := models.CurrentUser(c)

	if err := item.SafeDeleteOrInactivate(tx, user); err != nil {
		return reportError(c, err)
	}

	return c.Render(http.StatusNoContent, nil)
}

// convertItemApiInput creates a new `Item` from a `ItemInput`.
func convertItemApiInput(ctx context.Context, input api.ItemInput, policyID uuid.UUID) (models.Item, error) {
	item := models.Item{}
	if err := parseItemDates(input, &item); err != nil {
		return models.Item{}, err
	}

	item.Name = input.Name
	item.CategoryID = input.CategoryID
	item.InStorage = input.InStorage
	item.Country = input.Country
	item.Description = input.Description
	item.PolicyID = policyID
	item.Make = input.Make
	item.Model = input.Model
	item.SerialNumber = input.SerialNumber
	item.CoverageAmount = input.CoverageAmount
	item.CoverageStatus = input.CoverageStatus

	return item, nil
}

func parseItemDates(input api.ItemInput, modelItem *models.Item) error {
	pDate, err := time.Parse(domain.DateFormat, input.PurchaseDate)
	if err != nil {
		err = errors.New("failed to parse item purchase date, " + err.Error())
		appErr := api.NewAppError(err, api.ErrorItemInvalidPurchaseDate, api.CategoryUser)
		return appErr
	}
	modelItem.PurchaseDate = pDate

	csDate, err := time.Parse(domain.DateFormat, input.CoverageStartDate)
	if err != nil {
		err = errors.New("failed to parse item coverage start date, " + err.Error())
		appErr := api.NewAppError(err, api.ErrorItemInvalidCoverageStartDate, api.CategoryUser)
		return appErr
	}
	modelItem.CoverageStartDate = csDate

	return nil
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
