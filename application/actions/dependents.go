package actions

import (
	"errors"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/models"
)

func dependentsList(c buffalo.Context) error {
	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in route")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyNotFound, api.CategoryUser))
	}

	tx := models.Tx(c)
	policy.LoadDependents(tx, false)

	return renderOk(c, models.ConvertPolicyDependents(tx, policy.Dependents))
}

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
