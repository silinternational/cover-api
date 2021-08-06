package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

func claimsList(c buffalo.Context) error {
	tx := models.Tx(c)
	var claims models.Claims
	if err := tx.All(&claims); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(err))
	}

	return renderOk(c, models.ConvertClaims(claims))
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
