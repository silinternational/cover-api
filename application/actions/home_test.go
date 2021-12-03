package actions

import (
	"fmt"
	"net/http"

	"github.com/silinternational/cover-api/domain"
)

func (as *ActionSuite) Test_HomeHandler() {
	res := as.JSON("/").Get()

	as.Equal(http.StatusOK, res.Code)
	as.Contains(res.Body.String(), fmt.Sprintf("Welcome to %s API", domain.Env.AppName))
}
