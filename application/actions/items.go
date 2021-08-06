package actions

import (
	"errors"
	"net/http"

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
