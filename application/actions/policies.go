package actions

import (
	"errors"
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

// swagger:operation GET /policies Policies PoliciesList
//
// PoliciesList
//
// gets the data for all the user's Policies, or, if called by an admin, all Policies in the system
//
// ---
// responses:
//   '200':
//     description: all policies
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/Policy"
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

	apiPolicies := models.ConvertPolicies(tx, policies)

	return renderOk(c, apiPolicies)
}

func policiesListMine(c buffalo.Context) error {
	tx := models.Tx(c)
	user := models.CurrentUser(c)

	user.LoadPolicies(tx, false)

	apiPolicies := models.ConvertPolicies(tx, user.Policies)

	return renderOk(c, apiPolicies)
}

// swagger:operation PUT /policies/{id} Policies PoliciesUpdate
//
// PoliciesUpdate
//
// update a policy
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
//   - name: policy update input
//     in: body
//     description: policy update input object
//     required: true
//     schema:
//       "$ref": "#/definitions/PolicyUpdate"
// responses:
//   '200':
//     description: updated Policy
//     schema:
//       "$ref": "#/definitions/Policy"
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

	return renderOk(c, models.ConvertPolicy(tx, *policy))
}

// swagger:operation GET /policies/{id}/members PolicyMembers PolicyMembersList
//
// PolicyMembersList
//
// gets the data for all the members of a Policy
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
// responses:
//   '200':
//     description: all policy members
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/PolicyMember"
func policiesListMembers(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in context")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyFromContext, api.CategoryInternal))
	}

	policy.LoadMembers(tx, false)

	return renderOk(c, models.ConvertPolicyMembers(tx, policy.Members))
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
