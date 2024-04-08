package actions

import (
	"github.com/labstack/echo/v4"

	"github.com/silinternational/cover-api/api"
)

// swagger:operation GET /config/claim-incident-types Config ClaimIncidentTypes
// ClaimIncidentTypes
//
// list all valid Claim Incident Types
// ---
//
//	responses:
//	  '200':
//	    description: list of valid Claim Incident Types
//	    schema:
//	      type: array
//	      items:
//	        "$ref": "#/definitions/ClaimIncidentTypeStruct"
func claimIncidentTypes(c echo.Context) error {
	return renderOk(c, api.AllClaimIncidentTypes)
}

// swagger:operation GET /config/countries Config Countries
// Countries
//
// list of countries
// ---
//
//	responses:
//	  '200':
//	    description: list of countries
//	    schema:
//	      type: array
//	      items:
//	        "$ref": "#/definitions/Country"
func countries(c echo.Context) error {
	return renderOk(c, api.AllCountries)
}
