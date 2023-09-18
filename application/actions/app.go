// Cover API
//
//	Terms Of Service:
//	  there are no TOS at this moment, use at your own risk we take no responsibility
//
//	 Schemes: https
//	 Host: localhost
//	 BasePath: /
//	 Version: 0.0.1
//	 License: MIT http://opensource.org/licenses/MIT
//
//	 Consumes:
//	 - application/json
//
//	 Produces:
//	 - application/json
//
//	 Security:
//	 - oauth2:
//
//	 SecurityDefinitions:
//	 oauth2:
//	   type: oauth2
//	   authorizationUrl: /auth/login
//	   tokenUrl: /auth/token
//	   flow: implicit
//	   scopes:
//	     all: scopes are not used at this time
//
// swagger:meta
package actions

import (
	"fmt"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo-pop/v3/pop/popmw"
	"github.com/gobuffalo/logger"
	contenttype "github.com/gobuffalo/mw-contenttype"
	i18n "github.com/gobuffalo/mw-i18n/v2"
	paramlogger "github.com/gobuffalo/mw-paramlogger"
	"github.com/gorilla/sessions"
	"github.com/rs/cors"

	"github.com/silinternational/cover-api/auth"
	"github.com/silinternational/cover-api/job"
	"github.com/silinternational/cover-api/log"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/listeners"
	"github.com/silinternational/cover-api/locales"
	"github.com/silinternational/cover-api/models"
)

const idRegex = `/{id:[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[1-5][a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}}`

const (
	auditsPath          = "/audits"
	stewardPath         = "/steward"
	usersPath           = "/" + domain.TypeUser
	claimsPath          = "/" + domain.TypeClaim
	claimFilesPath      = "/" + domain.TypeClaimFile
	claimItemsPath      = "/" + domain.TypeClaimItem
	filesPath           = "/" + domain.TypeFile
	itemsPath           = "/" + domain.TypeItem
	ledgerReportPath    = "/" + domain.TypeLedgerReport
	policiesPath        = "/" + domain.TypePolicy
	policyDependentPath = "/" + domain.TypePolicyDependent
	entityCodesPath     = "/" + domain.TypeEntityCode
	policyMemberPath    = "/" + domain.TypePolicyMember
	repairsPath         = "/repairs"
	strikesPath         = "/" + domain.TypeStrike
)

var app *buffalo.App

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
		domain.Init()
		auth.Init()

		app = buffalo.New(buffalo.Options{
			Env:    domain.Env.GoEnv,
			Logger: logger.Logrus{FieldLogger: log.ErrLogger.LocalLog},
			PreWares: []buffalo.PreWare{
				cors.New(cors.Options{
					AllowCredentials: true,
					AllowedOrigins:   []string{domain.Env.UIURL},
					AllowedMethods:   []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
					AllowedHeaders:   []string{"*"},
				}).Handler,
			},
			SessionName:  "_cover_api_session",
			SessionStore: cookieStore(),
		})

		app.Use(hstsMiddleware)

		var err error
		domain.T, err = i18n.New(locales.FS(), "en")
		if err != nil {
			_ = app.Stop(err)
		}
		app.Use(domain.T.Middleware())

		registerCustomErrorHandler(app)

		// Initialize and attach service providers to context
		app.Use(log.SentryMiddleware)

		// Log request parameters (filters apply).
		app.Use(paramlogger.ParameterLogger)

		// Add authentication and authorization middleware
		app.Use(AuthN, AuthZ)
		app.Middleware.Skip(AuthN, HomeHandler, statusHandler)
		app.Middleware.Skip(AuthZ, HomeHandler, statusHandler, uploadHandler)

		// Set the request content type to JSON
		app.Use(contenttype.Set(domain.ContentJson))

		// Wraps each request in a transaction.
		app.Use(popmw.Transaction(models.DB))

		app.GET("/", HomeHandler)
		app.GET("/status", statusHandler)

		app.POST("/upload", uploadHandler)

		// users
		usersGroup := app.Group(usersPath)
		usersGroup.GET("/", usersList)
		usersGroup.Middleware.Skip(AuthZ, usersMe, usersMeUpdate, usersMeFilesAttach, usersMeFilesDelete)
		usersGroup.GET("/me", usersMe)
		usersGroup.PUT("/me", usersMeUpdate)
		usersGroup.POST("/me/files", usersMeFilesAttach)
		usersGroup.DELETE("/me/files", usersMeFilesDelete)
		usersGroup.GET(idRegex, usersView)

		auditsGroup := app.Group(auditsPath)
		auditsGroup.Middleware.Skip(AuthZ, auditRun) // AuthZ is implemented in the handler
		auditsGroup.POST("/", auditRun)

		auth := app.Group("/auth")
		auth.Middleware.Skip(AuthN, authRequest, authCallback, authDestroy)
		auth.Middleware.Skip(AuthZ, authRequest, authCallback, authDestroy)
		auth.POST("/login", authRequest)
		auth.POST("/callback", authCallback)
		auth.GET("/logout", authDestroy)

		// accounting ledger
		ledgerReportGroup := app.Group(ledgerReportPath)
		// AuthZ is implemented in the handlers
		ledgerReportGroup.Middleware.Skip(AuthZ, ledgerAnnualRenewalStatus, ledgerAnnualRenewalProcess)
		ledgerReportGroup.Middleware.Skip(AuthZ, ledgerMonthlyRenewalStatus, ledgerMonthlyRenewalProcess)
		ledgerReportGroup.GET("/", ledgerReportList)
		ledgerReportGroup.GET(idRegex, ledgerReportView)
		ledgerReportGroup.POST("/", ledgerReportCreate)
		ledgerReportGroup.PUT(idRegex, ledgerReportReconcile)
		ledgerReportGroup.GET("/annual", ledgerAnnualRenewalStatus)
		ledgerReportGroup.POST("/annual", ledgerAnnualRenewalProcess)
		ledgerReportGroup.GET("/monthly", ledgerMonthlyRenewalStatus)
		ledgerReportGroup.POST("/monthly", ledgerMonthlyRenewalProcess)

		stewardGroup := app.Group(stewardPath)
		stewardGroup.Middleware.Skip(AuthZ, stewardListRecentObjects) // AuthZ is implemented in the handler
		stewardGroup.GET("/"+api.ResourceRecent, stewardListRecentObjects)

		// claims
		claimsGroup := app.Group(claimsPath)
		claimsGroup.GET("/", claimsList)
		claimsGroup.GET(idRegex, claimsView)
		claimsGroup.PUT(idRegex, claimsUpdate)
		claimsGroup.DELETE(idRegex, claimsRemove)
		claimsGroup.POST(idRegex+filesPath, claimFilesAttach)
		claimsGroup.POST(idRegex+itemsPath, claimsItemsCreate)
		claimsGroup.POST(idRegex+"/"+api.ResourceSubmit, claimsSubmit)
		claimsGroup.POST(idRegex+"/"+api.ResourceRevision, claimsRequestRevision)
		claimsGroup.POST(idRegex+"/"+api.ResourcePreapprove, claimsPreapprove)
		claimsGroup.POST(idRegex+"/"+api.ResourceReceipt, claimsRequestReceipt)
		claimsGroup.POST(idRegex+"/"+api.ResourceApprove, claimsApprove)
		claimsGroup.POST(idRegex+"/"+api.ResourceDeny, claimsDeny)

		claimFilesGroup := app.Group(claimFilesPath)
		claimFilesGroup.DELETE(idRegex, claimFilesDelete)

		claimItemsGroup := app.Group(claimItemsPath)
		claimItemsGroup.PUT(idRegex, claimItemsUpdate)

		// config
		configGroup := app.Group("/config")
		configGroup.Middleware.Skip(AuthZ, claimIncidentTypes, itemCategoriesList, entityCodesList, countries)
		configGroup.GET("/countries", countries)
		configGroup.GET("/claim-incident-types", claimIncidentTypes)
		configGroup.GET("/item-categories", itemCategoriesList)

		// dependent
		depsGroup := app.Group(policyDependentPath)
		depsGroup.PUT(idRegex, dependentsUpdate)
		depsGroup.DELETE(idRegex, dependentsDelete)

		// entity codes
		entityCodesGroup := app.Group(entityCodesPath)
		entityCodesGroup.GET("", entityCodesList)
		entityCodesGroup.PUT(idRegex, entityCodesUpdate)
		entityCodesGroup.GET(idRegex, entityCodesView)
		entityCodesGroup.POST("", entityCodesCreate)

		// item
		itemsGroup := app.Group(itemsPath)
		itemsGroup.POST(idRegex+"/"+api.ResourceSubmit, itemsSubmit)
		itemsGroup.POST(idRegex+"/"+api.ResourceRevision, itemsRevision)
		itemsGroup.POST(idRegex+"/"+api.ResourceApprove, itemsApprove)
		itemsGroup.POST(idRegex+"/"+api.ResourceDeny, itemsDeny)
		itemsGroup.PUT(idRegex, itemsUpdate)
		itemsGroup.DELETE(idRegex, itemsRemove)

		// policies
		policiesGroup := app.Group(policiesPath)
		policiesGroup.GET("/", policiesList)
		policiesGroup.POST("/", policiesCreateTeam)
		policiesGroup.GET(idRegex, policiesView)
		policiesGroup.GET(idRegex+"/dependents", dependentsList)
		policiesGroup.PUT(idRegex, policiesUpdate)
		policiesGroup.POST(idRegex+"/dependents", dependentsCreate)
		policiesGroup.GET(idRegex+itemsPath, itemsList)
		policiesGroup.POST(idRegex+itemsPath, itemsCreate)
		policiesGroup.GET(idRegex+claimsPath, policiesClaimsList)
		policiesGroup.POST(idRegex+claimsPath, claimsCreate)
		policiesGroup.GET(idRegex+"/members", policiesListMembers)
		policiesGroup.POST(idRegex+"/members", policiesInviteMember)
		policiesGroup.POST(idRegex+"/ledger-reports", policiesLedgerReportCreate)
		policiesGroup.GET(idRegex+"/ledger-reports", policiesLedgerTableView)
		policiesGroup.POST(idRegex+"/"+api.ResourceStrikes, policiesStrikeCreate)

		// policy-members
		policyMembersGroup := app.Group(policyMemberPath)
		policyMembersGroup.DELETE(idRegex, policiesMembersDelete)

		// repairs
		repairsGroup := app.Group(repairsPath)
		repairsGroup.Middleware.Skip(AuthZ, repairsRun) // AuthZ is implemented in the handler
		repairsGroup.POST("/", repairsRun)

		// strikes
		strikesGroup := app.Group(strikesPath)
		strikesGroup.PUT(idRegex, strikesUpdate)
		strikesGroup.DELETE(idRegex, strikesDelete)

		listeners.RegisterListener()

		job.Init(&app.Worker)
	}

	return app
}

func cookieStore() sessions.Store {
	store := sessions.NewCookieStore([]byte(domain.Env.SessionSecret))

	store.Options.SameSite = http.SameSiteDefaultMode
	store.Options.HttpOnly = true

	if !domain.Env.DisableTLS {
		// Cookies will be sent in all contexts, i.e. in responses to both first-party and cross-origin requests.
		// This appears to be required to work with Firefox default cookie blocking setting.
		store.Options.SameSite = http.SameSiteNoneMode
		store.Options.Secure = true
	}

	return store
}

func hstsMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		headerValue := fmt.Sprintf("max-age=%d", domain.Env.HstsMaxAge)
		c.Response().Header().Add("Strict-Transport-Security", headerValue)
		return next(c)
	}
}
