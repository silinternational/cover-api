package actions

import (
	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/models"
)

func itemCategoriesList(c buffalo.Context) error {
	tx := models.Tx(c)

	var itemCategories models.ItemCategories
	if err := tx.Where("status = ?", api.ItemCategoryStatusEnabled).Order("name asc").All(&itemCategories); err != nil {
		return reportError(c, err)
	}

	apiCats := models.ConvertItemCategories(tx, itemCategories)

	return renderOk(c, apiCats)
}
