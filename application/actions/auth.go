package actions

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/auth"
	"github.com/silinternational/cover-api/auth/saml"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
	"github.com/silinternational/cover-api/models"
)

const (
	// http cookie access token
	AccessTokenSessionKey = "AccessToken"

	// http param for an auth invite code
	InviteCodeParam      = "invite"
	InviteCodeSessionKey = "invite_code"

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
	IDPPublicCert2:              replaceNewLines(domain.Env.SamlIdpCert2),
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
//	responses:
//	  '200':
//	    description: returns a "RedirectURL" key with the saml idp url that has a saml request
func authRequest(c buffalo.Context) error {

	getOrSetReturnTo(c)

	inviteCode := c.Param(InviteCodeParam)
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
func validateInviteOnLogin(c buffalo.Context, inviteCode string) *api.AppError {
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

	inviteCode, ok := c.Session().Get(InviteCodeSessionKey).(string)
	if ok {
		invite := models.PolicyUserInvite{}
		if err := invite.Accept(tx, inviteCode, user); err != nil {
			return reportErrorAndClearSession(c, err)
		}
	}

	// login was success, clear session so new login can be initiated if needed
	c.Session().Clear()

	isNew := false
	if time.Since(user.CreatedAt) < time.Duration(time.Second*30) {
		isNew = true
	}
	authUser.IsNew = isNew

	uat, err := user.CreateAccessToken(tx)
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
	log.SetUser(c, authUser.StaffID, user.GetName().String(), user.Email)

	// Set the authentication token in a cookie
	c.Session().Set(AccessTokenSessionKey, authUser.AccessToken)

	return c.Redirect(302, getLoginSuccessRedirectURL(*authUser, returnTo))
}

// getLoginSuccessRedirectURL generates the URL for redirection after a successful login
func getLoginSuccessRedirectURL(authUser auth.User, returnTo string) string {
	uiURL := domain.Env.UIURL
	params := ""
	if len(returnTo) > 0 {
		params = "?" + ReturnToParam + "=" + url.QueryEscape(returnTo)
	}

	// New Users go straight to the welcome page
	if authUser.IsNew {
		uiURL += "/welcome"
	}

	return uiURL + params
}

// swagger:operation GET /auth/logout Authentication AuthLogout
// AuthLogout
//
// Logout of application
// ---
//
//	responses:
//	  '302':
//	    description: redirect to UI
func authDestroy(c buffalo.Context) error {
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
		c.Session().Clear()
		redirectURL = authResp.RedirectURL
	}

	return c.Redirect(http.StatusFound, redirectURL)
}

func replaceNewLines(input string) string {
	return strings.Replace(input, `\n`, "\n", -1)
}
