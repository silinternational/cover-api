// Riskman API
//
// Terms Of Service:
//
// there are no TOS at this moment, use at your own risk we take no responsibility
//
//     Schemes: https
//     Host: localhost
//     BasePath: /
//     Version: 0.0.1
//     License: MIT http://opensource.org/licenses/MIT
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Security:
//     - oauth2:
//
//     SecurityDefinitions:
//     oauth2:
//         type: oauth2
//         authorizationUrl: /auth/login
//         tokenUrl: /auth/token
//         scopes:
//           all: scopes are not used at this time
//         flow: implicit
//
// swagger:meta
package actions

import (
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo-pop/v2/pop/popmw"
	"github.com/gobuffalo/envy"
	contenttype "github.com/gobuffalo/mw-contenttype"
	i18n "github.com/gobuffalo/mw-i18n"
	paramlogger "github.com/gobuffalo/mw-paramlogger"
	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/sessions"
	"github.com/rs/cors"

	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

const idRegex = `/{id:[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}}`

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var (
	ENV = envy.Get("GO_ENV", "development")
	app *buffalo.App
)

// App is where all routes and middleware for buffalo
// should be defined. This is the nerve center of your
// application.
//
// Routing, middleware, groups, etc... are declared TOP -> DOWN.
// This means if you add a middleware to `app` *after* declaring a
// group, that group will NOT have that new middleware. The same
// is true of resource declarations as well.
//
// It also means that routes are checked in the order they are declared.
// `ServeFiles` is a CATCH-ALL route, so it should always be
// placed last in the route declarations, as it will prevent routes
// declared after it to never be called.
func App() *buffalo.App {
	if app == nil {
		app = buffalo.New(buffalo.Options{
			Env: domain.Env.GoEnv,
			PreWares: []buffalo.PreWare{
				cors.New(cors.Options{
					AllowCredentials: true,
					AllowedOrigins:   []string{domain.Env.UIURL},
					AllowedMethods:   []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
					AllowedHeaders:   []string{"*"},
				}).Handler,
			},
			SessionName:  "_riskman_api_session",
			SessionStore: sessions.NewCookieStore([]byte(domain.Env.SessionSecret)),
		})

		var err error
		domain.T, err = i18n.New(packr.New("locales", "../locales"), "en")
		if err != nil {
			_ = app.Stop(err)
		}
		app.Use(domain.T.Middleware())

		registerCustomErrorHandler(app)

		// Initialize and attach "rollbar" to context
		app.Use(domain.RollbarMiddleware)

		// Log request parameters (filters apply).
		app.Use(paramlogger.ParameterLogger)

		//  Add authentication and authorization
		app.Use(AuthN, AuthZ)
		app.Middleware.Skip(AuthN, HomeHandler, statusHandler)
		app.Middleware.Skip(AuthZ, HomeHandler, statusHandler, itemCategoriesList)

		// Set the request content type to JSON
		app.Use(contenttype.Set("application/json"))

		// Wraps each request in a transaction.
		app.Use(popmw.Transaction(models.DB))

		app.GET("/", HomeHandler)
		app.GET("/status", statusHandler)
		app.GET("/item-categories", itemCategoriesList)

		// users
		usersGroup := app.Group("/" + domain.TypeUser)
		usersGroup.GET("/", usersList)
		usersGroup.Middleware.Skip(AuthZ, usersMe)
		usersGroup.GET("/me", usersMe)
		usersGroup.GET(idRegex, usersView)

		auth := app.Group("/auth")
		auth.Middleware.Skip(AuthN, authRequest, authCallback, authDestroy)
		auth.Middleware.Skip(AuthZ, authRequest, authCallback, authDestroy)
		auth.POST("/login", authRequest)
		auth.POST("/callback", authCallback)
		auth.GET("/logout", authDestroy)

		// claims
		claimsGroup := app.Group("/" + domain.TypeClaim)
		claimsGroup.GET("/", claimsList)
		claimsGroup.GET(idRegex, claimsView)

		// policies
		policiesGroup := app.Group("/" + domain.TypePolicy)
		policiesGroup.GET("/", policiesList)
		policiesGroup.GET(idRegex+"/dependents", dependentsList)
		policiesGroup.PUT(idRegex, policiesUpdate)
		policiesGroup.POST(idRegex+"/dependents", dependentsCreate)
		policiesGroup.GET(idRegex+"/items", itemsList)
		policiesGroup.POST(idRegex+"/items", itemsCreate)
		policiesGroup.POST(idRegex+"/claims", claimsCreate)
	}

	return app
}
