package actions

import (
	"context"
	"errors"
	"net/http"
	"time"

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

	apiItems := models.ConvertItems(tx, policy.Items)

	return c.Render(http.StatusOK, r.JSON(apiItems))
}

func itemsAdd(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in context")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyFromContext, api.CategoryInternal))
	}

	var itemPost api.ItemAddInput
	if err := StrictBind(c, &itemPost); err != nil {
		return reportError(c, err)
	}

	item, err := convertItemCreateInput(c, itemPost)
	if err != nil {
		return reportError(c, err)
	}

	if err := item.Create(tx); err != nil {
		return reportError(c, err)
	}

	output := models.ConvertItem(tx, item)

	return c.Render(http.StatusOK, r.JSON(output))
}

// convertItemCreateInput creates a new `Item` from a `ItemAddInput`.
func convertItemCreateInput(ctx context.Context, input api.ItemAddInput) (models.Item, error) {
	item := models.Item{}
	if err := parseItemDates(input, &item); err != nil {
		return models.Item{}, err
	}

	item.Name = input.Name
	item.CategoryID = input.CategoryID
	item.InStorage = input.InStorage
	item.Country = input.Country
	item.Description = input.Description
	item.PolicyID = input.PolicyID
	item.Make = input.Make
	item.Model = input.Model
	item.SerialNumber = input.SerialNumber
	item.CoverageAmount = input.CoverageAmount
	item.CoverageStatus = input.CoverageStatus

	return item, nil
}

func parseItemDates(input api.ItemAddInput, modelItem *models.Item) error {
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
