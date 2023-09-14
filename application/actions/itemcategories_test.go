package actions

import (
	"fmt"
	"net/http"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_ItemCategoriesList() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   1,
	}
	fixtures := models.CreatePolicyFixtures(as.DB, fixConfig)

	rc := models.RiskCategory{
		Name:       "Stationary",
		PolicyMax:  10000,
		CostCenter: "STATIONARY",
	}
	models.MustCreate(as.DB, &rc)

	// create 3 enabled categories
	cats := make(models.ItemCategories, 3)
	for i := range cats {
		cats[i] = models.ItemCategory{
			RiskCategoryID: rc.ID,
			Name:           fmt.Sprintf("Cat%v", i),
			Status:         api.ItemCategoryStatusEnabled,
			AutoApproveMax: 10,
			BillingPeriod:  12,
		}
		models.MustCreate(as.DB, &cats[i])
	}

	// create 1 disabled category
	disabled := models.ItemCategory{
		RiskCategoryID: rc.ID,
		Name:           "disabled",
		Status:         api.ItemCategoryStatusDisabled,
		AutoApproveMax: 100,
	}
	models.MustCreate(as.DB, &disabled)

	req := as.JSON("/config/item-categories")
	req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", fixtures.Policies[0].Members[0].Email)
	req.Headers["content-type"] = domain.ContentJson
	res := req.Get()

	body := res.Body.String()
	as.Equal(http.StatusOK, res.Code, "incorrect status code returned, body: %s", body)
	for i := range cats {
		as.Contains(body, cats[i].ID.String())
	}

	as.NotContains(body, disabled.ID.String())
}
