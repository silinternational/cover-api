package actions

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

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

	// http param and session key for Auth Email
	AuthEmailParam      = "auth-email"
	AuthEmailSessionKey = "AuthEmail"

	// http param and session key for Client ID
	ClientIDParam      = "client-id"
	ClientIDSessionKey = "ClientID"

	// logout http param for what is normally the bearer token
	LogoutToken = "token"

	// http param and session key for ReturnTo
	ReturnToParam      = "return-to"
	ReturnToSessionKey = "ReturnTo"
)

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
		username := strings.TrimSpace(user.FirstName + " " + user.LastName)
		domain.RollbarSetPerson(c, user.ID.String(), username, user.Email)
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

	// Get the AuthEmail param and push it into the Session
	authEmail := c.Param(AuthEmailParam)
	if authEmail == "" {
		authErr := authError{
			httpStatus: http.StatusBadRequest,
			errorKey:   api.ErrorMissingAuthEmail,
			errorMsg:   AuthEmailParam + " is required to login",
		}
		return authRequestError(c, authErr)
	}
	c.Session().Set(AuthEmailSessionKey, authEmail)

	getOrSetReturnTo(c)

	authConfig := saml.Config{}
	sp, err := saml.New(authConfig)
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

	authEmail, ok := c.Session().Get(AuthEmailSessionKey).(string)
	if !ok {
		return logErrorAndRedirect(c, api.ErrorMissingSessionAuthEmail,
			AuthEmailSessionKey+" session entry is required to complete login")
	}

	authConfig := saml.Config{}
	sp, err := saml.New(authConfig)
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

	tx := models.Tx(c)
	if err := user.FindOrCreateFromAuthUser(tx, authResp.AuthUser); err != nil {
		return logErrorAndRedirect(c, api.ErrorWithAuthUser, err.Error())
	}

	if inviteType != "" {
		dealWithInviteFromCallback(c, inviteType, objectUUID, user)
	}

	authUser, err := newOrgBasedAuthUser(c, clientID, user, org)
	if err != nil {
		return err
	}

	// set person on rollbar session
	domain.RollbarSetPerson(c, authUser.ID, authUser.Nickname, authUser.Email)

	return c.Redirect(302, getLoginSuccessRedirectURL(authUser, returnTo))
}

// Make extras variadic, so that it can be omitted from the params
func logErrorAndRedirect(c buffalo.Context, code api.ErrorKey, message string) error {
	domain.Error(c, message)

	c.Session().Clear()

	uiUrl := domain.Env.UIURL + "/login"
	return c.Redirect(http.StatusFound, uiUrl)
}
