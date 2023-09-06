package actions

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /policies/{id}/dependents PolicyDependents PolicyDependentsList
// PolicyDependentsList
//
// list policy dependents
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: policy ID
//	responses:
//	  '200':
//	    description: a list of PolicyDependents
//	    schema:
//	      type: array
//	      items:
//	        "$ref": "#/definitions/PolicyDependents"
func dependentsList(c buffalo.Context) error {
	policy := getReferencedPolicyFromCtx(c)

	tx := models.Tx(c)
	policy.LoadDependents(tx, false)

	return renderOk(c, policy.Dependents.ConvertToAPI())
}

// swagger:operation POST /policies/{id}/dependents PolicyDependents PolicyDependentsCreate
// PolicyDependentsCreate
//
// create a new PolicyDependent on a policy
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: policy ID
//	  - name: policy dependent
//	    in: body
//	    description: policy dependent input object
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/PolicyDependentInput"
//	responses:
//	  '200':
//	    description: the new PolicyDependent
//	    schema:
//	      "$ref": "#/definitions/PolicyDependent"
func dependentsCreate(c buffalo.Context) error {
	policy := getReferencedPolicyFromCtx(c)

	var input api.PolicyDependentInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)
	dependent, err := policy.AddDependent(tx, input)
	if err != nil {
		return reportError(c, err)
	}

	policy.LoadDependents(tx, false)

	return renderOk(c, dependent.ConvertToAPI())
}

// swagger:operation PUT /policy-dependents/{id} PolicyDependents PolicyDependentsUpdate
// PolicyDependentsUpdate
//
// update a policy dependent
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: policy dependent ID
//	  - name: policy dependent update input
//	    in: body
//	    description: policy dependent input object
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/PolicyDependentInput"
//	responses:
//	  '200':
//	    description: the updated PolicyDependent
//	    schema:
//	      "$ref": "#/definitions/PolicyDependent"
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

// swagger:operation DELETE /policy-dependents/{id} PolicyDependents PolicyDependentsDelete
// PolicyDependentsDelete
//
// Delete a policy dependent if it has no related policy items.
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: policy dependent ID
//	responses:
//	  '204':
//	    description: OK but no content in response
func dependentsDelete(c buffalo.Context) error {
	tx := models.Tx(c)
	dependent := getReferencedDependentFromCtx(c)

	relatedItemNames := dependent.RelatedItemNames(tx)
	if len(relatedItemNames) > 0 {
		err := errors.New("unable to delete policy dependent, since it is named on these items: " +
			strings.Join(relatedItemNames, "; "))

		appErr := api.NewAppError(err, api.ErrorPolicyDependentDelete, api.CategoryForbidden)
		appErr.HttpStatus = http.StatusConflict
		return reportError(c, appErr)
	}

	dependent.Destroy(tx)
	if err := dependent.Destroy(tx); err != nil {
		return reportError(c, err)
	}
	return c.Render(http.StatusNoContent, nil)
}

// getReferencedDependentFromCtx pulls the models.PolicyDependent resource from context that was put there
// by the AuthZ middleware
func getReferencedDependentFromCtx(c buffalo.Context) *models.PolicyDependent {
	dep, ok := c.Value(domain.TypePolicyDependent).(*models.PolicyDependent)
	if !ok {
		panic("policy dependent not found in context")
	}
	return dep
}
