package actions

import (
	"github.com/labstack/echo/v4"

	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /config/item-categories Config ItemCategoriesList
// ItemCategoriesList
//
// list all the enabled item categories
// ---
//
//	responses:
//	  '200':
//	    description: a list of ItemCategories
//	    schema:
//	      type: array
//	      items:
//	        "$ref": "#/definitions/ItemCategory"
func itemCategoriesList(c echo.Context) error {
	tx := models.Tx(c)

	var itemCategories models.ItemCategories
	if err := itemCategories.AllEnabled(tx); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, itemCategories.ConvertToAPI(tx))
}
