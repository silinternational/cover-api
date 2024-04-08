package actions

import (
	"fmt"
	"net/http"

	"github.com/silinternational/cover-api/domain"
)

func (as *ActionSuite) Test_HomeHandler() {
	body, status := as.request("GET", "/", "", nil)
	as.Equal(http.StatusOK, status)
	as.Contains(string(body), fmt.Sprintf("Welcome to %s API", domain.Env.AppName))
}
