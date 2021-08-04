package actions

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/silinternational/riskman-api/auth"

	"github.com/gobuffalo/buffalo/render"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/auth/saml"
	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

const (
	// http param for access token
	AccessTokenParam = "access-token"

	// http param and session key for Client ID
	ClientIDParam      = "client-id"
	ClientIDSessionKey = "ClientID"

	// logout http param for what is normally the bearer token
	LogoutToken = "token"

	// http param and session key for ReturnTo
	ReturnToParam      = "return-to"
	ReturnToSessionKey = "ReturnTo"

	// http param for token type
	TokenTypeParam = "token-type"
)

func replaceNewLines(input string) string {
	output := strings.Replace(input, `\n`, "\n", -1)
	return output
}

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

type authError struct {
	httpStatus int
	errorKey   api.ErrorKey
	errorMsg   string
}

// Make extras variadic, so that it can be omitted from the params
func authRequestError(c buffalo.Context, authErr authError, extras ...map[string]interface{}) error {
	domain.Error(c, authErr.errorMsg)

	appErr := api.AppError{
		HttpStatus: authErr.httpStatus,
		Key:        authErr.errorKey,
	}

	c.Session().Clear()

	return c.Render(authErr.httpStatus, render.JSON(appErr))
}

func setCurrentUser(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		bearerToken := domain.GetBearerTokenFromRequest(c.Request())
		if bearerToken == "" {
			return c.Error(http.StatusUnauthorized, errors.New("no Bearer token provided"))
		}

		var userAccessToken models.UserAccessToken
		tx := models.Tx(c)
		err := userAccessToken.FindByBearerToken(tx, bearerToken)
		if err != nil {
			if domain.IsOtherThanNoRows(err) {
				domain.Error(c, err.Error())
			}
			return c.Error(http.StatusUnauthorized, errors.New("invalid bearer token"))
		}

		isExpired, err := userAccessToken.DeleteIfExpired(tx)
		if err != nil {
			domain.Error(c, err.Error())
		}

		if isExpired {
			return c.Error(http.StatusUnauthorized, errors.New("expired bearer token"))
		}

		user, err := userAccessToken.GetUser(tx)
		if err != nil {
			return c.Error(http.StatusInternalServerError, fmt.Errorf("error finding user by access token, %s", err.Error()))
		}
		c.Set(domain.ContextKeyCurrentUser, user)

		// set person on rollbar session
		domain.RollbarSetPerson(c, user.ID.String(), user.FirstName, user.LastName, user.Email)
		msg := fmt.Sprintf("user %s authenticated with bearer token from ip %s", user.Email, c.Request().RemoteAddr)
		domain.NewExtra(c, "user_id", user.ID)
		domain.NewExtra(c, "email", user.Email)
		domain.NewExtra(c, "ip", c.Request().RemoteAddr)
		domain.Info(c, msg)

		return next(c)
	}
}

func authRequest(c buffalo.Context) error {
	// Push the Client ID into the Session
	clientID := c.Param(ClientIDParam)
	if clientID == "" {
		authErr := authError{
			httpStatus: http.StatusBadRequest,
			errorKey:   api.ErrorMissingClientID,
			errorMsg:   ClientIDParam + " is required to login",
		}
		return authRequestError(c, authErr)
	}

	c.Session().Set(ClientIDSessionKey, clientID)

	getOrSetReturnTo(c)

	sp, err := saml.New(samlConfig)
	if err != nil {
		return authRequestError(c, authError{
			httpStatus: http.StatusInternalServerError,
			errorKey:   api.ErrorLoadingAuthProvider,
			errorMsg:   "unable to load saml auth provider.",
		})
	}

	redirectURL, err := sp.AuthRequest(c)
	if err != nil {
		return authRequestError(c, authError{
			httpStatus: http.StatusInternalServerError,
			errorKey:   api.ErrorGettingAuthURL,
			errorMsg:   "unable to determine what the saml authentication url should be",
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

func authCallback(c buffalo.Context) error {
	clientID, ok := c.Session().Get(ClientIDSessionKey).(string)
	if !ok {
		return logErrorAndRedirect(c, api.ErrorMissingSessionClientID,
			ClientIDSessionKey+" session entry is required to complete login")
	}

	sp, err := saml.New(samlConfig)
	if err != nil {
		return authRequestError(c, authError{
			httpStatus: http.StatusInternalServerError,
			errorKey:   api.ErrorLoadingAuthProvider,
			errorMsg:   "unable to load saml auth provider in auth callback.",
		})
	}

	authResp := sp.AuthCallback(c)
	if authResp.Error != nil {
		return logErrorAndRedirect(c, api.ErrorAuthProvidersCallback, authResp.Error.Error())
	}

	returnTo := getOrSetReturnTo(c)

	if authResp.AuthUser == nil {
		return logErrorAndRedirect(c, api.ErrorAuthProvidersCallback, "nil authResp.AuthUser")
	}

	// if we have an authuser, find or create user in local db and finish login
	var user models.User

	// login was success, clear session so new login can be initiated if needed
	c.Session().Clear()

	authUser := authResp.AuthUser
	tx := models.Tx(c)
	if err := user.FindOrCreateFromAuthUser(tx, authUser); err != nil {
		return logErrorAndRedirect(c, api.ErrorWithAuthUser, err.Error())
	}

	isNew := false
	if time.Since(user.CreatedAt) < time.Duration(time.Second*30) {
		isNew = true
	}
	authUser.IsNew = isNew

	accessToken, expiresAt, err := user.CreateAccessToken(tx, clientID)
	if err != nil {
		return logErrorAndRedirect(c, api.ErrorCreatingAccessToken, err.Error())
	}

	authUser.AccessToken = accessToken
	authUser.AccessTokenExpiresAt = expiresAt

	// set person on rollbar session
	domain.RollbarSetPerson(c, user.StaffID, user.FirstName, user.LastName, user.Email)

	return c.Redirect(302, getLoginSuccessRedirectURL(*authUser, returnTo))
}

// Make extras variadic, so that it can be omitted from the params
func logErrorAndRedirect(c buffalo.Context, code api.ErrorKey, message string) error {
	domain.Error(c, message)

	c.Session().Clear()

	uiUrl := domain.Env.UIURL + "/login"
	return c.Redirect(http.StatusFound, uiUrl)
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

// authDestroy uses the bearer token to find the user's access token and
//  calls the appropriate provider's logout function.
func authDestroy(c buffalo.Context) error {
	tokenParam := c.Param(LogoutToken)
	if tokenParam == "" {
		return logErrorAndRedirect(c, api.ErrorMissingLogoutToken,
			LogoutToken+" is required to logout")
	}

	var uat models.UserAccessToken
	tx := models.Tx(c)
	err := uat.FindByBearerToken(tx, tokenParam)
	if err != nil {
		return logErrorAndRedirect(c, api.ErrorFindingAccessToken, err.Error())
	}

	authUser, err := uat.GetUser(tx)
	if err != nil {
		return logErrorAndRedirect(c, api.ErrorAuthProvidersLogout, err.Error())
	}

	// set person on rollbar session
	domain.RollbarSetPerson(c, authUser.ID.String(), authUser.FirstName, authUser.LastName, authUser.Email)

	sp, err := saml.New(samlConfig)
	if err != nil {
		return logErrorAndRedirect(c, api.ErrorLoadingAuthProvider, err.Error())
	}

	authResp := sp.Logout(c)
	if authResp.Error != nil {
		return logErrorAndRedirect(c, api.ErrorAuthProvidersLogout, authResp.Error.Error())
	}

	redirectURL := domain.Env.UIURL

	if authResp.RedirectURL != "" {
		var uat models.UserAccessToken
		err = uat.DeleteByBearerToken(tx, tokenParam)
		if err != nil {
			return logErrorAndRedirect(c, api.ErrorDeletingAccessToken, err.Error())
		}
		c.Session().Clear()
		redirectURL = authResp.RedirectURL
	}

	return c.Redirect(http.StatusFound, redirectURL)
}
