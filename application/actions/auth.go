package actions

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/auth"
	"github.com/silinternational/cover-api/auth/saml"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

const (
	// http param for access token
	AccessTokenParam = "access-token"

	// http param and session key for Client ID
	ClientIDParam      = "client-id"
	ClientIDSessionKey = "ClientID"

	// http param for an auth invite code
	InviteCodeParam      = "invite"
	InviteCodeSessionKey = "invite_code"

	// logout http param for what is normally the bearer token
	LogoutToken = "token"

	// http param and session key for ReturnTo
	ReturnToParam      = "return-to"
	ReturnToSessionKey = "ReturnTo"

	// http param for token type
	TokenTypeParam = "token-type"
)

var samlConfig = saml.Config{
	IDPEntityID:                 domain.Env.SamlIdpEntityId,
	SPEntityID:                  domain.Env.SamlSpEntityId,
	SingleSignOnURL:             domain.Env.SamlSsoURL,
	SingleLogoutURL:             domain.Env.SamlSloURL,
	AudienceURI:                 domain.Env.SamlAudienceUri,
	AssertionConsumerServiceURL: domain.Env.SamlAssertionConsumerServiceUrl,
	IDPPublicCert:               replaceNewLines(domain.Env.SamlIdpCert),
	SPPublicCert:                replaceNewLines(domain.Env.SamlSpCert),
	SPPrivateKey:                replaceNewLines(domain.Env.SamlSpPrivateKey),
	SignRequest:                 domain.Env.SamlSignRequest,
	CheckResponseSigning:        domain.Env.SamlCheckResponseSigning,
	AttributeMap:                nil,
}

// swagger:operation POST /auth/login Authentication AuthLogin
//
// AuthLogin
//
// Start the SAML login process
//
// ---
// parameters:
// - name: client-id
//   in: query
//   required: true
//   description: the user's client id
// responses:
//   '200':
//     description: returns a "RedirectURL" key with the saml idp url that has a saml request
func authRequest(c buffalo.Context) error {
	// Push the Client ID into the Session
	clientID := c.Param(ClientIDParam)
	if clientID == "" {
		appErr := api.AppError{
			HttpStatus: http.StatusBadRequest,
			Key:        api.ErrorMissingClientID,
			Message:    ClientIDParam + " is required to login",
		}
		return reportErrorAndClearSession(c, &appErr)
	}

	c.Session().Set(ClientIDSessionKey, clientID)

	getOrSetReturnTo(c)

	inviteCode := c.Param(InviteCodeParam)
	if inviteCode != "" {
		if appErr := validateInviteOnLogin(inviteCode, c); appErr != nil {
			return reportErrorAndClearSession(c, appErr)
		}
	}

	sp, err := saml.New(samlConfig)
	if err != nil {
		return reportErrorAndClearSession(c, &api.AppError{
			Err:        err,
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorLoadingAuthProvider,
			Message:    "unable to load saml auth provider.",
		})
	}

	redirectURL, err := sp.AuthRequest(c)
	if err != nil {
		return reportErrorAndClearSession(c, &api.AppError{
			Err:        err,
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorGettingAuthURL,
			Message:    "unable to determine what the saml authentication url should be",
		})
	}

	authRedirect := map[string]string{
		"RedirectURL": redirectURL,
	}

	// Reply with a 200 and leave it to the UI to do the redirect
	return c.Render(http.StatusOK, render.JSON(authRedirect))
}

func getOrSetReturnTo(c buffalo.Context) string {
	returnTo := c.Param(ReturnToParam)

	if returnTo == "" {
		var ok bool
		returnTo, ok = c.Session().Get(ReturnToSessionKey).(string)
		if !ok {
			returnTo = domain.DefaultUIPath
		}

		return returnTo
	}

	c.Session().Set(ReturnToSessionKey, returnTo)

	return returnTo
}

// check for valid matching invite and save the code to the Session
func validateInviteOnLogin(inviteCode string, c buffalo.Context) *api.AppError {
	appErr := api.AppError{Key: api.ErrorProcessingAuthInviteCode}

	tx, ok := c.Value(domain.ContextKeyTx).(*pop.Connection)
	if !ok {
		appErr.HttpStatus = http.StatusInternalServerError
		appErr.DebugMsg = "bad context database connection"
		return &appErr
	}

	codeUUID, err := uuid.FromString(inviteCode)
	if err != nil {
		appErr.HttpStatus = http.StatusBadRequest
		appErr.DebugMsg = fmt.Sprintf("invalid invite code: %s, %v", inviteCode, err)
		return &appErr
	}

	invite := models.PolicyUserInvite{}
	if err := invite.FindByID(tx, codeUUID); err != nil {
		appErr.HttpStatus = http.StatusNotFound
		appErr.DebugMsg = "error finding policy_user_invite code: " + err.Error()
		return &appErr
	}

	if err := invite.DestroyIfExpired(tx); err != nil {
		return &appErr
	}

	c.Session().Set(InviteCodeSessionKey, inviteCode)

	return nil
}

func authCallback(c buffalo.Context) error {
	clientID, ok := c.Session().Get(ClientIDSessionKey).(string)
	if !ok {
		appError := api.AppError{
			Key:        api.ErrorMissingSessionKey,
			DebugMsg:   ClientIDSessionKey + " session entry is required to complete login",
			HttpStatus: http.StatusFound,
		}
		return reportErrorAndClearSession(c, &appError)
	}

	sp, err := saml.New(samlConfig)
	if err != nil {
		return reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorLoadingAuthProvider,
			Message:    "unable to load saml auth provider in auth callback.",
		})
	}

	authResp := sp.AuthCallback(c)
	if authResp.Error != nil {
		reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorAuthProvidersCallback,
			Message:    authResp.Error.Error(),
		})
	}

	returnTo := getOrSetReturnTo(c)

	if authResp.AuthUser == nil {
		reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusFound,
			Key:        api.ErrorAuthProvidersCallback,
			Message:    "nil authResp.AuthUser",
		})
	}

	// if we have an authuser, find or create user in local db and finish login
	var user models.User

	// login was success, clear session so new login can be initiated if needed
	c.Session().Clear()

	authUser := authResp.AuthUser
	tx := models.Tx(c)
	if err := user.FindOrCreateFromAuthUser(tx, authUser); err != nil {
		reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorWithAuthUser,
			Message:    err.Error(),
		})
	}

	inviteCode, ok := c.Session().Get(InviteCodeSessionKey).(string)
	if ok {
		invite := models.PolicyUserInvite{}
		if err := invite.Accept(tx, inviteCode, user); err != nil {
			reportErrorAndClearSession(c, err)
		}
	}

	isNew := false
	if time.Since(user.CreatedAt) < time.Duration(time.Second*30) {
		isNew = true
	}
	authUser.IsNew = isNew

	uat, err := user.CreateAccessToken(tx, clientID)
	if err != nil {
		reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorCreatingAccessToken,
			Message:    err.Error(),
		})
	}

	authUser.AccessToken = uat.AccessToken
	authUser.AccessTokenExpiresAt = uat.ExpiresAt.UTC().Unix()

	// set person on rollbar session
	domain.RollbarSetPerson(c, user.StaffID, user.FirstName, user.LastName, user.Email)

	return c.Redirect(302, getLoginSuccessRedirectURL(*authUser, returnTo))
}

// getLoginSuccessRedirectURL generates the URL for redirection after a successful login
func getLoginSuccessRedirectURL(authUser auth.User, returnTo string) string {
	uiURL := domain.Env.UIURL

	params := fmt.Sprintf("?%s=Bearer&%s=%s",
		TokenTypeParam, AccessTokenParam, authUser.AccessToken)

	// New Users go straight to the welcome page
	if authUser.IsNew {
		uiURL += "/welcome"
		if len(returnTo) > 0 {
			params += "&" + ReturnToParam + "=" + url.QueryEscape(returnTo)
		}
		return uiURL + params
	}

	// Avoid two question marks in the params
	if strings.Contains(returnTo, "?") && strings.HasPrefix(params, "?") {
		params = "&" + params[1:]
	}

	return uiURL + returnTo + params
}

// swagger:operation GET /auth/logout Authentication AuthLogout
//
// AuthLogout
//
// Logout of application
//
// ---
// parameters:
// - name: token
//   in: query
//   required: true
//   description: the user's bearer token
// responses:
//   '302':
//     description: redirect to UI
func authDestroy(c buffalo.Context) error {
	tokenParam := c.Param(LogoutToken)
	if tokenParam == "" {
		return reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusBadRequest,
			Key:        api.ErrorMissingLogoutToken,
			Message:    LogoutToken + " is required to logout",
		})
	}

	var uat models.UserAccessToken
	tx := models.Tx(c)
	if appErr := uat.FindByBearerToken(tx, tokenParam); appErr != nil {
		return reportErrorAndClearSession(c, appErr)
	}

	authUser, err := uat.GetUser(tx)
	if err != nil {
		return reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorAuthProvidersLogout,
			Message:    err.Error(),
		})
	}

	// set person on rollbar session
	domain.RollbarSetPerson(c, authUser.ID.String(), authUser.FirstName, authUser.LastName, authUser.Email)

	sp, err := saml.New(samlConfig)
	if err != nil {
		return reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorLoadingAuthProvider,
			Message:    err.Error(),
		})
	}

	authResp := sp.Logout(c)
	if authResp.Error != nil {
		return reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorAuthProvidersLogout,
			Message:    authResp.Error.Error(),
		})
	}

	redirectURL := domain.LogoutRedirectURL

	if authResp.RedirectURL != "" {
		var uat models.UserAccessToken
		if appErr := uat.DeleteByBearerToken(tx, tokenParam); appErr != nil {
			return reportErrorAndClearSession(c, appErr)
		}
		c.Session().Clear()
		redirectURL = authResp.RedirectURL
	}

	return c.Redirect(http.StatusFound, redirectURL)
}

func replaceNewLines(input string) string {
	return strings.Replace(input, `\n`, "\n", -1)
}

func checkSamlConfig() {
	if domain.Env.GoEnv == "development" || domain.Env.GoEnv == "test" {
		return
	}
	if domain.Env.SamlIdpEntityId == "" {
		panic("required SAML variable SamlIdpEntityId is undefined")
	}
	if domain.Env.SamlSpEntityId == "" {
		panic("required SAML variable SamlSpEntityId is undefined")
	}
	if domain.Env.SamlSsoURL == "" {
		panic("required SAML variable SamlSsoURL is undefined")
	}
	if domain.Env.SamlSloURL == "" {
		panic("required SAML variable SamlSloURL is undefined")
	}
	if domain.Env.SamlAudienceUri == "" {
		panic("required SAML variable SamlAudienceUri is undefined")
	}
	if domain.Env.SamlAssertionConsumerServiceUrl == "" {
		panic("required SAML variable SamlAssertionConsumerServiceUrl is undefined")
	}
	if domain.Env.SamlIdpCert == "" {
		panic("required SAML variable SamlIdpCert is undefined")
	}
	if domain.Env.SamlSpCert == "" {
		panic("required SAML variable SamlSpCert is undefined")
	}
	if domain.Env.SamlSpPrivateKey == "" {
		panic("required SAML variable SamlSpPrivateKey is undefined")
	}
}
