package domain

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/gobuffalo/envy"

	"github.com/gofrs/uuid"

	"github.com/gobuffalo/buffalo"
	mwi18n "github.com/gobuffalo/mw-i18n"
	"github.com/gobuffalo/packr/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/rollbar/rollbar-go"
)

var (
	// Logger is a plain instance of log.Logger, normally set to stdout
	Logger log.Logger

	// ErrLogger is an instance of ErrLogProxy, and is the only error logging
	// mechanism that can be used without access to the Buffalo context.
	ErrLogger ErrLogProxy

	AuthCallbackURL string
)

// T is the Buffalo i18n translator
var T *mwi18n.Translator

// Assets is a packr box with asset files such as images
var Assets *packr.Box

var extrasLock = sync.RWMutex{}

// BuffaloContextType is a custom type used as a value key passed to context.WithValue as per the recommendations
// in the function docs for that function: https://golang.org/pkg/context/#WithValue
type BuffaloContextType string

// BuffaloContext is the key for the call to context.WithValue in gqlHandler
const BuffaloContext = BuffaloContextType("BuffaloContext")

// Context keys
const (
	ContextKeyCurrentUser = "current_user"
	ContextKeyExtras      = "extras"
	ContextKeyRollbar     = "rollbar"
)

func getBuffaloContext(ctx context.Context) buffalo.Context {
	bc, ok := ctx.Value(BuffaloContext).(buffalo.Context)
	if ok {
		return bc
	}

	// Doesn't have a BuffaloContext value, so it must be the actual BuffaloContext
	return ctx.(buffalo.Context)
}

// Env Holds the values of environment variables
var Env struct {
	GoEnv         string `ignored:"true"`
	ApiBaseURL    string `required:"true" split_words:"true"`
	AppName       string `default:"riskman" split_words:"true"`
	SessionSecret string `required:"true" split_words:"true"`
	ServerRoot    string `default:"" split_words:"true"`
	RollbarToken  string `default:"" split_words:"true"`
	UIURL         string `default:"missing.ui.url"`
}

func init() {
	readEnv()
	Logger.SetOutput(os.Stdout)
	ErrLogger.SetOutput(os.Stderr)
	ErrLogger.InitRollbar()
	Assets = packr.New("Assets", "../assets")
	AuthCallbackURL = Env.ApiBaseURL + "/auth/callback"
}

// readEnv loads environment data into `Env`
func readEnv() {
	err := envconfig.Process("riskman", &Env)
	if err != nil {
		log.Fatal(errors.New("error loading env vars: " + err.Error()))
	}

	// Doing this separately to avoid needing two environment variables for the same thing
	Env.GoEnv = envy.Get("GO_ENV", "development")
}

// ErrLogProxy is a "tee" that sends to Rollbar and to the local logger,
// normally set to stderr. Rollbar is disabled if `GoEnv` is "test", and
// is a client instantiation separate from the one used in the Rollbar
// middleware.
type ErrLogProxy struct {
	LocalLog  log.Logger
	RemoteLog *rollbar.Client
}

func (e *ErrLogProxy) SetOutput(w io.Writer) {
	e.LocalLog.SetOutput(w)
}

func (e *ErrLogProxy) Printf(format string, a ...interface{}) {
	// Send to local logger
	e.LocalLog.Printf(format, a...)

	// Only send to remote log if not in test env
	if Env.GoEnv == "test" {
		return
	}
	e.RemoteLog.Errorf(rollbar.ERR, format, a...)
}

func (e *ErrLogProxy) InitRollbar() {
	e.RemoteLog = rollbar.New(
		Env.RollbarToken,
		Env.GoEnv,
		"",
		"",
		Env.ServerRoot)
}

// NewExtra Sets a new key-value pair in the `extras` entry of the context
func NewExtra(ctx context.Context, key string, e interface{}) {
	c := getBuffaloContext(ctx)
	extras := getExtras(c)

	extrasLock.Lock()
	defer extrasLock.Unlock()
	extras[key] = e

	c.Set(ContextKeyExtras, extras)
}

func getExtras(c buffalo.Context) map[string]interface{} {
	extras, _ := c.Value(ContextKeyExtras).(map[string]interface{})
	if extras == nil {
		extras = map[string]interface{}{}
	}

	return extras
}

// GetUUID creates a new, unique version 4 (random) UUID and returns it
// as a uuid2.UUID. Errors are ignored.
func GetUUID() uuid.UUID {
	id, err := uuid.NewV4()
	if err != nil {
		ErrLogger.Printf("error creating new uuid ... %v", err)
	}
	return id
}

func RollbarMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		if Env.RollbarToken == "" || Env.GoEnv == "test" {
			return next(c)
		}

		client := rollbar.New(
			Env.RollbarToken,
			Env.GoEnv,
			"",
			"",
			Env.ServerRoot)
		defer client.Close()

		c.Set(ContextKeyRollbar, client)

		return next(c)
	}
}

// GetBearerTokenFromRequest obtains the token from an Authorization header beginning
// with "Bearer". If not found, an empty string is returned.
func GetBearerTokenFromRequest(r *http.Request) string {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return ""
	}

	re := regexp.MustCompile(`^(?i)Bearer (.*)$`)
	matches := re.FindSubmatch([]byte(authorizationHeader))
	if len(matches) < 2 {
		return ""
	}

	return string(matches[1])
}
