package actions

import (
	"fmt"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /steward/recent Steward ListRecentObjects
//
// ListRecentObjects
//
// gets Items and Claims that have recently had their coverage_status/status changed
//
// ---
//	responses:
//	  '200':
//	    description: a list of Items and a list of Claims which each have the time when their status was last changed.
//	    schema:
//	      "$ref": "#/definitions/RecentObjects"
func stewardListRecentObjects(c buffalo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("actor not allowed to perform that action on this resource")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	tx := models.Tx(c)

	items, err := models.ItemsWithRecentStatusChanges(tx)
	if err != nil {
		return reportError(c, err)
	}

	claims, err := models.ClaimsWithRecentStatusChanges(tx)
	if err != nil {
		return reportError(c, err)
	}

	recent := api.RecentObjects{
		Items:  items,
		Claims: claims,
	}

	return renderOk(c, recent)
}
