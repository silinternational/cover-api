package actions

import (
	"errors"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/models"
)

// swagger:operation GET /policies/{id}/dependents PolicyDependents PolicyDependentsList
//
// PolicyDependentsList
//
// list policy dependents
//
// ---
// responses:
//   '200':
//     description: a list of PolicyDependents
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/PolicyDependents"
func dependentsList(c buffalo.Context) error {
	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in route")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyNotFound, api.CategoryInternal))
	}

	tx := models.Tx(c)
	policy.LoadDependents(tx, false)

	return renderOk(c, models.ConvertPolicyDependents(tx, policy.Dependents))
}

// swagger:operation POST /policies/{id}/dependents PolicyDependents PolicyDependentsCreate
//
// PolicyDependentsCreate
//
// create a new PolicyDependent on a policy
//
// ---
// parameters:
//   - name: policy dependent
//     in: body
//     description: policy dependent input object
//     required: true
//     schema:
//       "$ref": "#/definitions/PolicyDependentInput"
// responses:
//   '200':
//     description: the new PolicyDependent
//     schema:
//       "$ref": "#/definitions/PolicyDependent"
func dependentsCreate(c buffalo.Context) error {
	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in route")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyNotFound, api.CategoryUser))
	}

	var input api.PolicyDependentInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorPolicyDependentCreateInvalidInput, api.CategoryUser))
	}

	tx := models.Tx(c)
	if err := policy.AddDependent(tx, input); err != nil {
		return reportError(c, err)
	}

	policy.LoadDependents(tx, false)

	return renderOk(c, models.ConvertPolicyDependents(tx, policy.Dependents))
}
