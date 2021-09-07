package domain

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
	mwi18n "github.com/gobuffalo/mw-i18n"
	"github.com/gobuffalo/packr/v2"
	"github.com/gofrs/uuid"
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

var AllowedFileUploadTypes = []string{
	"image/bmp",
	"image/gif",
	"image/jpeg",
	"image/png",
	"image/webp",
	"application/pdf",
}

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

	DefaultUIPath = "/home"

	EventPayloadID = "id"

	TypeClaim           = "claims"
	TypeClaimItem       = "claim-items"
	TypeFile            = "files"
	TypeItem            = "items"
	TypePolicy          = "policies"
	TypePolicyDependent = "policy-dependents"
	TypePolicyUser      = "policy-users"
	TypeUser            = "users"
)

const (
	CurrencyFactor = 100
	DateFormat     = "2006-01-02"

	// How many hours old an item can be until it's not allowed to be deleted
	ItemDeleteCutOffHours = 72

	MaxFileSize = 1024 * 1024 * 10 // 10 Megabytes

	DurationDay  = time.Duration(time.Hour * 24)
	DurationWeek = time.Duration(DurationDay * 7)
	Megabyte     = 1048576
)

// Event Kinds
const (
	EventApiUserCreated   = "api:user:created"
	EventApiItemSubmitted = "api:item:submitted"
	EventApiItemRevision  = "api:item:revision"
	EventApiItemApproved  = "api:item:approved"
	EventApiItemDenied    = "api:item:denied"

	EventApiClaimReview1     = "api:claim:review1"
	EventApiClaimRevision    = "api:claim:revision"
	EventApiClaimPreapproved = "api:claim:preapproved"
	EventApiClaimReceipt     = "api:claim:receipt"
	EventApiClaimReview2     = "api:claim:review2"
	EventApiClaimReview3     = "api:claim:review3"
	EventApiClaimApproved    = "api:claim:approved"
	EventApiClaimDenied      = "api:claim:denied"
)

//  Notification templates
const (
	MessageTemplateItemSubmittedSteward = "item_submitted_steward"
	MessageTemplateItemApprovedMember   = "item_approved_member"
	MessageTemplateItemAutoSteward      = "item_auto_approved_steward"
	MessageTemplateItemRevisionMember   = "item_revision_member"
	MessageTemplateItemDeniedMember     = "item_denied_member"
)

// redirect url for after logout
var LogoutRedirectURL = "missing.ui.url/logged-out"

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
	GoEnv                      string `ignored:"true"`
	ApiBaseURL                 string `required:"true" split_words:"true"`
	AccessTokenLifetimeSeconds int    `default:"1166400" split_words:"true"` // 13.5 days
	AppName                    string `default:"Cover" split_words:"true"`

	ListenerDelayMilliseconds int `default:"1000" split_words:"true"`
	ListenerMaxRetries        int `default:"10" split_words:"true"`

	SessionSecret     string `required:"true" split_words:"true"`
	RollbarServerRoot string `default:"" split_words:"true"`
	RollbarToken      string `default:"" split_words:"true"`
	UIURL             string `default:"http://missing.ui.url"`

	SamlSpEntityId                  string `default:"" split_words:"true"`
	SamlAudienceUri                 string `default:"" split_words:"true"`
	SamlIdpEntityId                 string `default:"" split_words:"true"`
	SamlIdpCert                     string `default:"" split_words:"true"`
	SamlSpCert                      string `default:"" split_words:"true"`
	SamlSpPrivateKey                string `default:"" split_words:"true"`
	SamlAssertionConsumerServiceUrl string `default:"" split_words:"true"`
	SamlSsoURL                      string `default:"" split_words:"true"`
	SamlSloURL                      string `default:"" split_words:"true"`
	SamlCheckResponseSigning        bool   `default:"true" split_words:"true"`
	SamlSignRequest                 bool   `default:"true" split_words:"true"`
	SamlRequireEncryptedAssertion   bool   `default:"true" split_words:"true"`

	AwsRegion          string `split_words:"true"`
	AwsS3Endpoint      string `split_words:"true"`
	AwsS3DisableSSL    bool   `split_words:"true"`
	AwsS3Bucket        string `split_words:"true"`
	AwsAccessKeyID     string `split_words:"true"`
	AwsSecretAccessKey string `split_words:"true"`
	EmailFromAddress   string `required:"true" split_words:"true"`
	EmailService       string `default:"ses" split_words:"true"`

	MaxFileDelete int `default:"10" split_words:"true"`

	// Ensure these reflect cents and not just dollars
	PolicyMaxCoverage       int `default:"50000" split_words:"true"` // will be multiplied by CurrencyFactor in readEnv()
	DependantAutoApproveMax int `default:"4000" split_words:"true"`  // will be multiplied by CurrencyFactor in readEnv()
}

func init() {
	readEnv()
	Logger.SetOutput(os.Stdout)
	ErrLogger.SetOutput(os.Stderr)
	ErrLogger.InitRollbar()
	Assets = packr.New("Assets", "../assets")
	AuthCallbackURL = Env.ApiBaseURL + "/auth/callback"

	LogoutRedirectURL = Env.UIURL + "/logged-out"
}

// readEnv loads environment data into `Env`
func readEnv() {
	err := envconfig.Process("", &Env)
	if err != nil {
		log.Fatal(errors.New("error loading env vars: " + err.Error()))
	}

	Env.PolicyMaxCoverage *= 100
	Env.DependantAutoApproveMax *= 100

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
			Env.RollbarServerRoot)
		defer client.Close()

		c.Set(ContextKeyRollbar, client)

		return next(c)
	}
}

// EmailFromAddress combines a name with the configured from address for use in an email From header. If name is nil,
// only the App Name will be used.
func EmailFromAddress(name *string) string {
	addr := Env.AppName + " <" + Env.EmailFromAddress + ">"
	if name != nil {
		addr = *name + " via " + addr
	}
	return addr
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

// IsOtherThanNoRows returns false if the error is nil or is just reporting that there
//   were no rows in the result set for a sql query.
func IsOtherThanNoRows(err error) bool {
	if err == nil {
		return false
	}

	if strings.Contains(err.Error(), sql.ErrNoRows.Error()) {
		return false
	}

	return true
}

// RollbarSetPerson sets person on the rollbar context for further logging
func RollbarSetPerson(c buffalo.Context, id, userFirst, userLast, email string) {
	username := strings.TrimSpace(userFirst + " " + userLast)
	rc, ok := c.Value(ContextKeyRollbar).(*rollbar.Client)
	if ok {
		rc.SetPerson(id, username, email)
	}
}

func MergeExtras(extras []map[string]interface{}) map[string]interface{} {
	allExtras := map[string]interface{}{}

	// I didn't think I would need this, but without it at least one test was failing
	// The code allowed a map[string]interface{} to get through (i.e. not in a slice)
	// without the compiler complaining
	if len(extras) == 1 {
		return extras[0]
	}

	for _, e := range extras {
		for k, v := range e {
			allExtras[k] = v
		}
	}

	return allExtras
}

// IsStringInSlice iterates over a slice of strings, looking for the given
// string. If found, true is returned. Otherwise, false is returned.
func IsStringInSlice(needle string, haystack []string) bool {
	for _, hs := range haystack {
		if needle == hs {
			return true
		}
	}

	return false
}

func RandomString(n int, includeLetters string) string {
	if includeLetters == "" {
		includeLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}
	letters := []rune(includeLetters)
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] // #nosec G404
	}
	return string(b)
}
