package actions

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation PUT /strikes/{id} Strikes StrikesUpdate
// StrikesUpdate
//
// update a strike
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: strike ID
//	  - name: strike input
//	    in: body
//	    description: policy item update object
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/StrikeInput"
//	responses:
//	  '200':
//	    description: updated Strike
//	    schema:
//	      "$ref": "#/definitions/Strike"
func strikesUpdate(c echo.Context) error {
	strike := getReferencedStrikeFromCtx(c)

	var input api.StrikeInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	strike.Description = input.Description

	if err := strike.Update(c); err != nil {
		return reportError(c, err)
	}

	output := strike.ConvertToAPI()
	return c.JSON(http.StatusOK, output)
}

// swagger:operation DELETE /strikes/{id} Strikes StrikesDelete
// StrikesDelete
//
// Delete a strike.
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: item ID
//	responses:
//	  '204':
//	    description: OK but no content in response
func strikesDelete(c echo.Context) error {
	strike := getReferencedStrikeFromCtx(c)

	if err := strike.Destroy(models.Tx(c)); err != nil {
		return reportError(c, err)
	}

	return c.JSON(http.StatusNoContent, nil)
}

// getReferencedStrikeFromCtx pulls the models.Strike resource from context that was put there
// by the AuthZ middleware
func getReferencedStrikeFromCtx(c echo.Context) *models.Strike {
	strike, ok := c.Get(domain.TypeStrike).(*models.Strike)
	if !ok {
		panic("strike not found in context")
	}
	return strike
}
