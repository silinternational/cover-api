package actions

import (
	"errors"
	"net/http"

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

	return renderOk(c, apiPolicies)
}

func policiesListMine(c buffalo.Context) error {
	tx := models.Tx(c)
	user := models.CurrentUser(c)

	user.LoadPolicies(tx, false)

	apiPolicies, err := models.ConvertPolicies(tx, user.Policies)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	return renderOk(c, apiPolicies)
}

func policiesUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in context")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyFromContext, api.CategoryInternal))
	}

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

	return renderOk(c, apiPolicy)
}

func policiesListMembers(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in context")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyFromContext, api.CategoryInternal))
	}

	policy.LoadMembers(tx, false)

	members, err := models.ConvertPolicyMembers(tx, policy.Members)
	if err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorFailedToConvertToAPIType, api.CategoryInternal))
	}

	return renderOk(c, members)
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
