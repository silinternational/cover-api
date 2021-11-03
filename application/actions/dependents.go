package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
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

	tx := models.Tx(c)
	policy.LoadDependents(tx, false)

	return renderOk(c, policy.Dependents.ConvertToAPI())
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

	var input api.PolicyDependentInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)
	if err := policy.AddDependent(tx, input); err != nil {
		return reportError(c, err)
	}

	policy.LoadDependents(tx, false)

	return renderOk(c, policy.Dependents.ConvertToAPI())
}

// swagger:operation PUT /policy-dependents/{id} PolicyDependents PolicyDependentsUpdate
//
// PolicyDependentsUpdate
//
// update a policy dependent
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy dependent ID
//   - name: policy dependent update input
//     in: body
//     description: policy dependent create update object
//     required: true
//     schema:
//       "$ref": "#/definitions/PolicyDependentInput"
// responses:
//   '200':
//     description: the updated PolicyDependent
//     schema:
//       "$ref": "#/definitions/PolicyDependent"
func dependentsUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	dependent := getReferencedDependentFromCtx(c)

	var input api.PolicyDependentInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	dependent.Name = input.Name
	dependent.Relationship = input.Relationship
	dependent.Country = input.Country
	dependent.ChildBirthYear = input.ChildBirthYear

	if err := dependent.Update(tx); err != nil {
		return reportError(c, err)
	}

	output := dependent.ConvertToAPI()
	return c.Render(http.StatusOK, r.JSON(output))
}

// getReferencedDependentFromCtx pulls the models.Item resource from context that was put there
// by the AuthZ middleware
func getReferencedDependentFromCtx(c buffalo.Context) *models.PolicyDependent {
	dep, ok := c.Value(domain.TypePolicyDependent).(*models.PolicyDependent)
	if !ok {
		panic("policy dependent not found in context")
	}
	return dep
}
