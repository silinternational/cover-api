package actions

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/models"
)

func itemsList(c buffalo.Context) error {
	tx := models.Tx(c)

	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in context")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyFromContext, api.CategoryInternal))
	}

	err := policy.LoadItems(tx, true)
	if err != nil {
		appErr := api.NewAppError(err, api.ErrorPolicyLoadingItems, api.CategoryInternal)
		return reportError(c, appErr)
	}

	return renderOk(c, models.ConvertItems(tx, policy.Items))
}

func itemsAdd(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in context")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyFromContext, api.CategoryInternal))
	}

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

func itemsUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	item := getReferencedItemFromCtx(c)
	if item == nil {
		err := errors.New("item not found in context")
		return reportError(c, api.NewAppError(err, api.ErrorItemFromContext, api.CategoryInternal))
	}

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

	if err := newItem.Update(tx); err != nil {
		return reportError(c, err)
	}

	output := models.ConvertItem(tx, newItem)
	return c.Render(http.StatusOK, r.JSON(output))
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
		return nil
	}
	return item
}
