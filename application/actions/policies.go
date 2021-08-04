package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

func policiesList(c buffalo.Context) error {
	user := models.CurrentUser(c)

	if user.IsAdmin() {
		return policiesListAll(c)
	}

	return policiesListMine(c)
}

func policiesListAll(c buffalo.Context) error {
	tx := models.Tx(c)
	var policies models.Policies
	if err := tx.All(&policies); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	apiPolicies, err := models.ConvertPolicies(tx, policies)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(apiPolicies))
}

func policiesListMine(c buffalo.Context) error {
	tx := models.Tx(c)
	user := models.CurrentUser(c)

	if err := user.LoadPolicies(tx, false); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	apiPolicies, err := models.ConvertPolicies(tx, user.Policies)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(apiPolicies))
}

// getPolicyFromCtx pulls the models.Policy resource from context that was put there
// by the AuthZ middleware based on a URL pattern of /policy/{id}
func getPolicyFromCtx(c buffalo.Context) *models.Policy {
	policy, ok := c.Value(domain.TypePolicy).(*models.Policy)
	if !ok {
		return nil
	}
	return policy
}
