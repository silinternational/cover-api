package actions

import (
	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /policies/{id}/dependents PolicyDependents PolicyDependentsList
//
// PolicyDependentsList
//
// list policy dependents
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
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
		panic("policy not found in context")
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
//   - name: id
//     in: path
//     required: true
//     description: policy ID
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
		panic("policy not found in context")
	}

	var input api.PolicyDependentInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)
	if err := policy.AddDependent(tx, input); err != nil {
		return reportError(c, err)
	}

	policy.LoadDependents(tx, false)

	return renderOk(c, models.ConvertPolicyDependents(tx, policy.Dependents))
}
