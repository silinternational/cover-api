package actions

import (
	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /countries/{code} Countries CountriesByCode
// Countries Find By Code
//
// find a country by its ISO 3166-1 alpha-2 code
// ---
//
//	parameters:
//	  - name: code
//	    in: path
//	    required: true
//	    description: country code (ISO 3166-1 alpha-2)
//	responses:
//	  '200':
//	    description: country data
//	    schema:
//	      "$ref": "#/definitions/Country"
func countriesByCode(c buffalo.Context) error {
	code := c.Params().Get("code")
	var country models.Country
	err := country.FindByCode(models.Tx(c), code)
	if err != nil {
		return reportError(c, err)
	}
	return renderOk(c, &country)
}

// swagger:operation GET /countries Countries CountriesList
// CountriesList
//
// list all countries
// ---
//
//	responses:
//	  '200':
//	    description: country data
//	    schema:
//	      type: array
//	      items:
//	        "$ref": "#/definitions/Country"
func countriesList(c buffalo.Context) error {
	var countries models.Countries
	err := countries.All(models.Tx(c))
	if err != nil {
		return reportError(c, err)
	}
	return renderOk(c, &countries)
}
