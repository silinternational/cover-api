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
		return reportError(c, api.NewAppError(err, api.ErrorGettingPolicyFromContext, api.CategoryInternal))

	}

	err := policy.LoadItems(tx, true)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	apiItems, err := models.ConvertItems(tx, policy.Items)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(apiItems))
}
