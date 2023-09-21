package actions

import (
	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
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
func claimIncidentTypes(c buffalo.Context) error {
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
func countries(c buffalo.Context) error {
	return renderOk(c, api.AllCountries)
}

// swagger:operation GET /config/risk-categories Config RiskCategoriesList
// RiskCategoriesList
//
// list all the risk categories
// ---
//
//	responses:
//	  '200':
//	    description: a list of Risk Categories
//	    schema:
//	      type: array
//	      risks:
//	        "$ref": "#/definitions/RiskCategory"
func riskCategoriesList(c buffalo.Context) error {
	tx := models.Tx(c)

	var riskCategories models.RiskCategories
	if err := riskCategories.All(tx); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, riskCategories.ConvertToAPI(tx))
}
