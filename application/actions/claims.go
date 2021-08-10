package actions

import (
	"errors"
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

func claimsList(c buffalo.Context) error {
	tx := models.Tx(c)

	// TODO: list only current user's claims
	var claims models.Claims
	if err := tx.All(&claims); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	return renderOk(c, models.ConvertClaims(claims))
}

func claimsView(c buffalo.Context) error {
	claim := getReferencedClaimFromCtx(c)
	if claim == nil {
		err := errors.New("claim not found in context")
		return reportError(c, api.NewAppError(err, "", api.CategoryInternal))
	}
	return renderOk(c, models.ConvertClaim(*claim))
}

func claimsCreate(c buffalo.Context) error {
	policy := getReferencedPolicyFromCtx(c)
	if policy == nil {
		err := errors.New("policy not found in route")
		return reportError(c, api.NewAppError(err, api.ErrorPolicyNotFound, api.CategoryUser))
	}

	var input api.ClaimCreateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorClaimCreateInvalidInput, api.CategoryUser))
	}

	tx := models.Tx(c)
	if err := policy.AddClaim(tx, input); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, models.ConvertPolicy(tx, *policy))
}

// getReferencedClaimFromCtx pulls the models.Claim resource from context that was put there
// by the AuthZ middleware
func getReferencedClaimFromCtx(c buffalo.Context) *models.Claim {
	claim, ok := c.Value(domain.TypeClaim).(*models.Claim)
	if !ok {
		return nil
	}
	return claim
}
