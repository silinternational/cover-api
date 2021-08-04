package actions

import (
	"errors"

	"github.com/gobuffalo/buffalo"
	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/models"
)

func dependentsList(c buffalo.Context) error {
	policy := getPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in route")
		return reportError(c, api.NewAppError(err, "key", api.CategoryUser))

	}

	tx := models.Tx(c)
	if err := policy.LoadDependents(tx, false); err != nil {
		return reportError(c, api.NewAppError(err, "key", api.CategoryInternal))
	}

	return ok(c, policy.Dependents)
}
