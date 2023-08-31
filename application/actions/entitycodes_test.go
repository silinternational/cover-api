package actions

import (
	"fmt"
	"net/http"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_entityCodesList() {
	inactiveCode := models.EntityCode{Code: "ABC", Name: "ABC Code", Active: false, IncomeAccount: "67890"}
	as.NoError(as.DB.Create(&inactiveCode))
	activeCode := models.EntityCode{Code: "XYZ", Name: "XYZ Code", Active: true, IncomeAccount: "12345"}
	as.NoError(as.DB.Create(&activeCode))

	user := models.CreateUserFixtures(as.DB, 1).Users[0]
	admin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name       string
		actor      models.User
		wantStatus int
		wantError  api.ErrorKey
		wantCodes  []string
	}{
		{
			name:       "must be authenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantError:  api.ErrorNotAuthorized,
		},
		{
			name:       "any user can list",
			actor:      user,
			wantStatus: http.StatusOK,
			wantCodes:  []string{"XYZ"},
		},
		{
			name:       "admin sees all",
			actor:      admin,
			wantStatus: http.StatusOK,
			wantCodes:  []string{"ABC", "XYZ"},
		},
	}
	for _, tt := range tests {
		req := as.JSON(entityCodesPath)
		req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
		res := req.Get()
		body := res.Body.Bytes()

		as.Equal(tt.wantStatus, res.Code, "incorrect status code returned: %d\n%s", res.Code, body)
		if tt.wantError != "" {
			var err api.AppError
			as.NoError(as.decodeBody(body, &err), "response data is not as expected")
			as.Equal(tt.wantError, err.Key, "error key is incorrect")
		} else {
			var codes api.EntityCodes
			as.NoError(as.decodeBody(body, &codes), "response data is not as expected %s", body)

			as.Len(codes, len(tt.wantCodes))
			var gotCodes []string
			for _, c := range codes {
				gotCodes = append(gotCodes, c.Code)
			}
			as.ElementsMatch(tt.wantCodes, gotCodes)

			if tt.actor.IsAdmin() {
				as.NotNil(codes[0].IncomeAccount)
				as.NotNil(codes[0].ParentEntity)
			} else {
				as.Nil(codes[0].IncomeAccount)
				as.Nil(codes[0].ParentEntity)
			}
		}

	}
}

func (as *ActionSuite) Test_entityCodesUpdate() {
	inactiveCode := models.EntityCode{Code: "ABC", Name: "ABC Code", Active: false, IncomeAccount: "67890"}
	as.NoError(as.DB.Create(&inactiveCode))
	activeCode := models.EntityCode{Code: "XYZ", Name: "XYZ Code", Active: true, IncomeAccount: "12345"}
	as.NoError(as.DB.Create(&activeCode))

	user := models.CreateUserFixtures(as.DB, 1).Users[0]
	admin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name       string
		actor      models.User
		wantStatus int
		wantError  api.ErrorKey
	}{
		{
			name:       "must be authenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantError:  api.ErrorNotAuthorized,
		},
		{
			name:       "regular user cannot update codes",
			actor:      user,
			wantStatus: http.StatusNotFound,
			wantError:  api.ErrorNotAuthorized,
		},
		{
			name:       "admin can update",
			actor:      admin,
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		req := as.JSON("%s/%s", entityCodesPath, inactiveCode.ID)
		req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
		input := api.EntityCodeInput{
			Active:        true,
			IncomeAccount: "newacct",
			ParentEntity:  activeCode.Code,
		}
		res := req.Put(input)
		body := res.Body.Bytes()

		as.Equal(tt.wantStatus, res.Code, "incorrect status code returned: %d\n%s", res.Code, body)
		if tt.wantError != "" {
			var err api.AppError
			as.NoError(as.decodeBody(body, &err), "response data is not as expected", body)
			as.Equal(tt.wantError, err.Key, "error key is incorrect")
		} else {
			var code api.EntityCode
			as.NoError(as.decodeBody(body, &code), "response data is not as expected %s", body)
			as.NotNil(code.Active)
			as.NotNil(code.IncomeAccount)
			as.NotNil(code.ParentEntity)
			as.Equal(input.Active, *code.Active)
			as.Equal(input.IncomeAccount, *code.IncomeAccount)
			as.Equal(input.ParentEntity, *code.ParentEntity)
		}
	}
}

func (as *ActionSuite) Test_entityCodesView() {
	inactiveCode := models.EntityCode{Code: "ABC", Name: "ABC Code", Active: false, IncomeAccount: "67890"}
	as.NoError(as.DB.Create(&inactiveCode))
	activeCode := models.EntityCode{Code: "XYZ", Name: "XYZ Code", Active: true, IncomeAccount: "12345"}
	as.NoError(as.DB.Create(&activeCode))

	user := models.CreateUserFixtures(as.DB, 1).Users[0]
	admin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name       string
		actor      models.User
		wantStatus int
		wantError  api.ErrorKey
	}{
		{
			name:       "must be authenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantError:  api.ErrorNotAuthorized,
		},
		{
			name:       "regular user cannot view",
			actor:      user,
			wantStatus: http.StatusNotFound,
			wantError:  api.ErrorNotAuthorized,
		},
		{
			name:       "admin can view",
			actor:      admin,
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		req := as.JSON("%s/%s", entityCodesPath, inactiveCode.ID)
		req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
		res := req.Get()
		body := res.Body.Bytes()

		as.Equal(tt.wantStatus, res.Code, "incorrect status code returned: %d\n%s", res.Code, body)
		if tt.wantError != "" {
			var err api.AppError
			as.NoError(as.decodeBody(body, &err), "response data is not as expected", body)
			as.Equal(tt.wantError, err.Key, "error key is incorrect")
		} else {
			var code api.EntityCode
			as.NoError(as.decodeBody(body, &code), "response data is not as expected %s", body)
			as.NotNil(code.Active)
			as.NotNil(code.IncomeAccount)
			as.NotNil(code.ParentEntity)
			as.Equal(inactiveCode.Active, *code.Active)
			as.Equal(inactiveCode.IncomeAccount, *code.IncomeAccount)
			as.Equal(inactiveCode.ParentEntity, *code.ParentEntity)
		}
	}
}

// test entityCodesCreate
func (as *ActionSuite) Test_entityCodesCreate() {
	user := models.CreateUserFixtures(as.DB, 1).Users[0]
	admin := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		code       string
		name       string
		actor      models.User
		wantStatus int
		wantError  api.ErrorKey
	}{
		{
			name:       "must be authenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantError:  api.ErrorNotAuthorized,
		},
		{
			name:       "regular user cannot create entity codes",
			actor:      user,
			wantStatus: http.StatusNotFound,
			wantError:  api.ErrorNotAuthorized,
		},
		{
			name:       "admin can create",
			actor:      admin,
			wantStatus: http.StatusOK,
		},
	}
	input := api.EntityCodeCreateInput{
		Code:          "ABC",
		Name:          "ABC Code",
		Active:        true,
		IncomeAccount: "67890",
		ParentEntity:  "XYZ",
	}
	for _, tt := range tests {
		req := as.JSON("%s", entityCodesPath)
		req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
		res := req.Post(input)
		body := res.Body.Bytes()

		as.Equal(tt.wantStatus, res.Code, "incorrect status code returned: %d\n%s", res.Code, body)
		if tt.wantError != "" {
			var err api.AppError
			as.NoError(as.decodeBody(body, &err), "response data is not as expected", body)
			as.Equal(tt.wantError, err.Key, "error key is incorrect")
		} else {
			var code api.EntityCode
			as.NoError(as.decodeBody(body, &code), "response data is not as expected %s", body)
			as.NotNil(code.Active)
			as.NotNil(code.IncomeAccount)
			as.NotNil(code.ParentEntity)
			as.Equal(input.Active, *code.Active)
			as.Equal(input.IncomeAccount, *code.IncomeAccount)
			as.Equal(input.ParentEntity, *code.ParentEntity)
		}
	}
}
