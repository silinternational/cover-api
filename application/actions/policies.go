package actions

import (
	"net/http"

	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
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

func policiesUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)

	var update api.PolicyUpdate
	if err := StrictBind(c, &update); err != nil {
		return reportError(c, err)
	}

	switch policy.Type {
	case api.PolicyTypeHousehold:
		policy.HouseholdID = update.HouseholdID
		policy.CostCenter = ""
		policy.Account = ""
		policy.EntityCode = ""
	case api.PolicyTypeOU:
		policy.HouseholdID = ""
		policy.CostCenter = update.CostCenter
		policy.Account = update.Account
		policy.EntityCode = update.EntityCode
	}

	if err := policy.Update(tx); err != nil {
		return reportError(c, err)
	}

	apiPolicy, err := models.ConvertPolicy(tx, *policy)
	if err != nil {
		return reportError(c, err)
	}

	return c.Render(http.StatusOK, r.JSON(apiPolicy))
}

// getReferencedPolicyFromCtx pulls the models.Policy resource from context that was put there
// by the AuthZ middleware
func getReferencedPolicyFromCtx(c buffalo.Context) *models.Policy {
	policy, ok := c.Value(domain.TypePolicy).(*models.Policy)
	if !ok {
		return nil
	}
	return policy
}

func itemsList(c buffalo.Context) error {
	tx := models.Tx(c)

	cPolicy := c.Value(domain.TypePolicy)
	if cPolicy == nil {
		return c.Render(http.StatusInternalServerError, r.String("failed to find policy in context after authn"))
	}

	policy, ok := cPolicy.(models.Policy)
	if !ok {
		return c.Render(http.StatusInternalServerError, r.String("failed to convert context policy in policy model"))
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
