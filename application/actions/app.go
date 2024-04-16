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
	"errors"
	"net/http"
	"os"

	"github.com/gobuffalo/pop/v6"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/auth"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/listeners"
	"github.com/silinternational/cover-api/log"
	"github.com/silinternational/cover-api/models"
)

const idParam = `/:id`

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

var app *echo.Echo

func App() *echo.Echo {
	if app == nil {
		domain.Init()
		auth.Init()
		models.PatchItemCategories()

		app = echo.New()
		/*
			buffalo.Options{
					Env:    domain.Env.GoEnv,
					Logger: logger.Logrus{FieldLogger: log.ErrLogger.LocalLog},
					SessionName:  "_cover_api_session",
					SessionStore: cookieStore(),
				}
		*/

		app.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowCredentials: true,
			AllowOrigins:     []string{domain.Env.UIURL},
			AllowMethods:     []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
			AllowHeaders:     []string{"Authorization", "Content-Type"},
		}))

		// Logger Middleware
		app.Use(middleware.Logger())

		// Recover Middleware
		app.Use(middleware.Recover())

		// Session Middleware
		app.Use(session.Middleware(sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))))

		// DB Transaction Middleware
		app.Use(GetTxMiddleware(models.DB))

		if os.Getenv("GO_ENV") == "development" {
			app.Debug = true
		}

		// Initialize and attach service providers to context
		app.Use(log.SentryMiddleware)

		// Add authentication and authorization middleware
		app.Use(AuthN, AuthZ)

		app.GET("/", HomeHandler)
		app.GET("/status", statusHandler)

		app.POST("/upload", uploadHandler)

		// users
		usersGroup := app.Group(usersPath)
		usersGroup.GET("", usersList)
		usersGroup.GET("/me", usersMe)
		usersGroup.PUT("/me", usersMeUpdate)
		usersGroup.POST("/me/files", usersMeFilesAttach)
		usersGroup.DELETE("/me/files", usersMeFilesDelete)
		usersGroup.GET(idParam, usersView)

		auditsGroup := app.Group(auditsPath)
		auditsGroup.POST("", auditRun)

		authGroup := app.Group("/auth")
		authGroup.POST("/login", authRequest)
		authGroup.POST("/callback", authCallback)
		authGroup.GET("/logout", authDestroy)

		// accounting ledger
		ledgerReportGroup := app.Group(ledgerReportPath)
		// AuthZ is implemented in the handlers
		ledgerReportGroup.GET("", ledgerReportList)
		ledgerReportGroup.GET(idParam, ledgerReportView)
		ledgerReportGroup.POST("", ledgerReportCreate)
		ledgerReportGroup.PUT(idParam, ledgerReportReconcile)
		ledgerReportGroup.GET("/annual", ledgerAnnualRenewalStatus)
		ledgerReportGroup.POST("/annual", ledgerAnnualRenewalProcess)
		ledgerReportGroup.GET("/monthly", ledgerMonthlyRenewalStatus)
		ledgerReportGroup.POST("/monthly", ledgerMonthlyRenewalProcess)

		app.GET(stewardPath+"/"+api.ResourceRecent, stewardListRecentObjects)

		// claims
		claimsGroup := app.Group(claimsPath)
		claimsGroup.GET("", claimsList)
		claimsGroup.GET(idParam, claimsView)
		claimsGroup.PUT(idParam, claimsUpdate)
		claimsGroup.DELETE(idParam, claimsRemove)
		claimsGroup.POST(idParam+filesPath, claimFilesAttach)
		claimsGroup.POST(idParam+itemsPath, claimsItemsCreate)
		claimsGroup.POST(idParam+"/"+api.ResourceSubmit, claimsSubmit)
		claimsGroup.POST(idParam+"/"+api.ResourceRevision, claimsRequestRevision)
		claimsGroup.POST(idParam+"/"+api.ResourcePreapprove, claimsPreapprove)
		claimsGroup.POST(idParam+"/"+api.ResourceReceipt, claimsRequestReceipt)
		claimsGroup.POST(idParam+"/"+api.ResourceApprove, claimsApprove)
		claimsGroup.POST(idParam+"/"+api.ResourceDeny, claimsDeny)

		claimFilesGroup := app.Group(claimFilesPath)
		claimFilesGroup.DELETE(idParam, claimFilesDelete)

		claimItemsGroup := app.Group(claimItemsPath)
		claimItemsGroup.PUT(idParam, claimItemsUpdate)

		// config
		configGroup := app.Group("/config")
		configGroup.GET("/countries", countries)
		configGroup.GET("/claim-incident-types", claimIncidentTypes)
		configGroup.GET("/item-categories", itemCategoriesList)

		// dependent
		depsGroup := app.Group(policyDependentPath)
		depsGroup.PUT(idParam, dependentsUpdate)
		depsGroup.DELETE(idParam, dependentsDelete)

		// entity codes
		entityCodesGroup := app.Group(entityCodesPath)
		entityCodesGroup.GET("", entityCodesList)
		entityCodesGroup.PUT(idParam, entityCodesUpdate)
		entityCodesGroup.GET(idParam, entityCodesView)
		entityCodesGroup.POST("", entityCodesCreate)

		// item
		itemsGroup := app.Group(itemsPath)
		itemsGroup.POST(idParam+"/"+api.ResourceSubmit, itemsSubmit)
		itemsGroup.POST(idParam+"/"+api.ResourceRevision, itemsRevision)
		itemsGroup.POST(idParam+"/"+api.ResourceApprove, itemsApprove)
		itemsGroup.POST(idParam+"/"+api.ResourceDeny, itemsDeny)
		itemsGroup.PUT(idParam, itemsUpdate)
		itemsGroup.DELETE(idParam, itemsRemove)

		// policies
		policiesGroup := app.Group(policiesPath)
		policiesGroup.GET("", policiesList)
		policiesGroup.POST("", policiesCreateTeam)
		policiesGroup.POST("/import", policiesImport)
		policiesGroup.GET(idParam, policiesView)
		policiesGroup.GET(idParam+"/dependents", dependentsList)
		policiesGroup.PUT(idParam, policiesUpdate)
		policiesGroup.POST(idParam+"/dependents", dependentsCreate)
		policiesGroup.GET(idParam+itemsPath, itemsList)
		policiesGroup.POST(idParam+itemsPath, itemsCreate)
		policiesGroup.GET(idParam+claimsPath, policiesClaimsList)
		policiesGroup.POST(idParam+claimsPath, claimsCreate)
		policiesGroup.GET(idParam+"/members", policiesListMembers)
		policiesGroup.POST(idParam+"/members", policiesInviteMember)
		policiesGroup.POST(idParam+"/ledger-reports", policiesLedgerReportCreate)
		policiesGroup.GET(idParam+"/ledger-reports", policiesLedgerTableView)
		policiesGroup.POST(idParam+"/"+api.ResourceStrikes, policiesStrikeCreate)

		// policy-members
		policyMembersGroup := app.Group(policyMemberPath)
		policyMembersGroup.DELETE(idParam, policiesMembersDelete)

		// repairs
		repairsGroup := app.Group(repairsPath)
		repairsGroup.POST("", repairsRun)

		// strikes
		strikesGroup := app.Group(strikesPath)
		strikesGroup.PUT(idParam, strikesUpdate)
		strikesGroup.DELETE(idParam, strikesDelete)

		// robots
		app.GET("/robots.txt", robots)

		listeners.RegisterListener()

		routes := app.Routes()
		for _, r := range routes {
			log.Debugf("%s %s\n", r.Method, r.Path)
		}

		// FIXME
		// job.Init(&app.Worker)
	}

	return app
}

// DELETE?
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

func GetTxMiddleware(tx *pop.Connection) echo.MiddlewareFunc {
	errNotOK := errors.New("http error, rolling back transaction")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if tx == nil {
				// TODO: should this return an error?
				return next(c)
			}
			err := tx.Transaction(func(tx *pop.Connection) error {
				const key = "tx"
				c.Set(key, tx)

				if err := next(c); err != nil {
					return err
				}

				// If the status is not a "success", roll back transaction by returning an error
				res := c.Response()

				// let 200s and 300s through
				if res.Status < 200 || res.Status >= 400 {
					return errNotOK
				}

				return nil
			})
			if err != nil {
				if errors.Is(err, errNotOK) {
					return nil
				}
				return err
			}
			return nil
		}
	}
}
