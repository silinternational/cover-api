package domain

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sync"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
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
	ApiBaseURL        string `ignored:"true"`  // This will be set in readEnv based on the `HOST` env var
	AppName           string `default:"Riskman"`
	GoEnv             string `default:"development"`
	RollbarServerRoot string `required:"true"`
	RollbarToken      string `default:""`
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
	Env.ApiBaseURL = envy.Get("HOST", "")
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
		Env.RollbarServerRoot)
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
