package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/silinternational/riskman-api/models"
)

func policiesList(c buffalo.Context) error {
	user := getUserFromCxt(c)

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
	user := getUserFromCxt(c)

	if err := user.LoadPolicies(tx, false); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	apiPolicies, err := models.ConvertPolicies(tx, user.Policies)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	return c.Render(http.StatusOK, r.JSON(apiPolicies))
}
