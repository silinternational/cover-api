package actions

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/auth"
	"github.com/silinternational/cover-api/auth/saml"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
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
// AuthLogin
//
// Start the SAML login process
// ---
//
//	parameters:
//	- name: client-id
//	  in: query
//	  required: true
//	  description: the user's client id
//	responses:
//	  '200':
//	    description: returns a "RedirectURL" key with the saml idp url that has a saml request
func authRequest(c echo.Context) error {
	// Push the Client ID into the Session
	clientID := c.QueryParam(ClientIDParam)
	if clientID == "" {
		appErr := api.AppError{
			HttpStatus: http.StatusBadRequest,
			Key:        api.ErrorMissingClientID,
			Message:    ClientIDParam + " is required to login",
		}
		return reportErrorAndClearSession(c, &appErr)
	}

	err := sessionSetValue(c, ClientIDSessionKey, clientID)
	if err != nil {
		return err
	}

	getOrSetReturnTo(c)

	inviteCode := c.QueryParam(InviteCodeParam)
	if inviteCode != "" {
		if appErr := validateInviteOnLogin(c, inviteCode); appErr != nil {
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
	return c.JSON(http.StatusOK, authRedirect)
}

func getOrSetReturnTo(c echo.Context) string {
	returnTo := c.QueryParam(ReturnToParam)

	if returnTo == "" {
		var err error
		returnTo, err = sessionGetString(c, ReturnToSessionKey)
		if err != nil {
			returnTo = domain.DefaultUIPath
		}

		return returnTo
	}

	if err := sessionSetValue(c, ReturnToSessionKey, returnTo); err != nil {
		log.Errorf("failed to set %s in session: %s", ReturnToSessionKey, err)
	}

	return returnTo
}

// check for valid matching invite and save the code to the Session
func validateInviteOnLogin(c echo.Context, inviteCode string) *api.AppError {
	appErr := api.AppError{Key: api.ErrorProcessingAuthInviteCode}

	tx, ok := c.Get(domain.ContextKeyTx).(*pop.Connection)
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

	if err = sessionSetValue(c, InviteCodeSessionKey, inviteCode); err != nil {
		log.Errorf("failed to set %s in session: %s", InviteCodeSessionKey, err)
	}

	return nil
}

func authCallback(c echo.Context) error {
	clientID, err := sessionGetString(c, ClientIDSessionKey)
	if err != nil {
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
		err = fmt.Errorf("auth response error: %w", authResp.Error)
		return reportErrorAndClearSession(c, api.NewAppError(err, api.ErrorAuthProvidersCallback, api.CategoryInternal))
	}

	returnTo := getOrSetReturnTo(c)

	if authResp.AuthUser == nil {
		return reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusFound,
			Key:        api.ErrorAuthProvidersCallback,
			Err:        errors.New("nil authResp.AuthUser"),
		})
	}

	// if we have an authuser, find or create user in local db and finish login
	var user models.User

	authUser := authResp.AuthUser
	tx := models.Tx(c)
	if err := user.FindOrCreateFromAuthUser(tx, authUser); err != nil {
		return reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorWithAuthUser,
			Message:    err.Error(),
		})
	}

	inviteCode, err := sessionGetString(c, InviteCodeSessionKey)
	if err == nil {
		invite := models.PolicyUserInvite{}
		if err := invite.Accept(tx, inviteCode, user); err != nil {
			return reportErrorAndClearSession(c, err)
		}
	}

	// login was success, clear session so new login can be initiated if needed
	if err = clearSession(c); err != nil {
		return reportError(c, appErrorFromErr(err))
	}

	isNew := false
	if time.Since(user.CreatedAt) < time.Duration(time.Second*30) {
		isNew = true
	}
	authUser.IsNew = isNew

	uat, err := user.CreateAccessToken(tx, clientID)
	if err != nil {
		return reportErrorAndClearSession(c, &api.AppError{
			HttpStatus: http.StatusInternalServerError,
			Key:        api.ErrorCreatingAccessToken,
			Message:    err.Error(),
		})
	}

	authUser.AccessToken = uat.AccessToken
	authUser.AccessTokenExpiresAt = uat.ExpiresAt.UTC().Unix()

	// set person on log context
	log.SetUser(c.Request().Context(), authUser.StaffID, user.GetName().String(), user.Email)

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
// AuthLogout
//
// Logout of application
// ---
//
//	parameters:
//	- name: token
//	  in: query
//	  required: true
//	  description: the user's bearer token
//	responses:
//	  '302':
//	    description: redirect to UI
func authDestroy(c echo.Context) error {
	tokenParam := c.QueryParam(LogoutToken)
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

	// set person on log context
	log.SetUser(c.Request().Context(), authUser.ID.String(), authUser.GetName().String(), authUser.Email)

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
		if err = clearSession(c); err != nil {
			return reportError(c, appErrorFromErr(err))
		}
		redirectURL = authResp.RedirectURL
	}

	return c.Redirect(http.StatusFound, redirectURL)
}

func replaceNewLines(input string) string {
	return strings.Replace(input, `\n`, "\n", -1)
}
