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
		CostCenter: "STATIONARY",
	}
	models.MustCreate(as.DB, &rc)

	// create 3 enabled categories
	cats := make(models.ItemCategories, 3)
	for i := range cats {
		cats[i] = models.ItemCategory{
			Key:            fmt.Sprintf("Key%v", i),
			RiskCategoryID: rc.ID,
			Name:           fmt.Sprintf("Cat%v", i),
			Status:         api.ItemCategoryStatusEnabled,
			AutoApproveMax: 10,
			BillingPeriod:  domain.BillingPeriodAnnual,
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

	uat, uatErr := fixtures.Policies[0].Members[0].CreateAccessToken(as.DB)
	as.NoError(uatErr)
	as.Session.Set(AccessTokenSessionKey, uat.AccessToken)
	req := as.JSON("/config/item-categories")
	req.Headers["content-type"] = domain.ContentJson
	res := req.Get()

	body := res.Body.String()
	as.Equal(http.StatusOK, res.Code, "incorrect status code returned, body: %s", body)
	for i := range cats {
		as.Contains(body, cats[i].ID.String())
	}

	as.NotContains(body, disabled.ID.String())
}
