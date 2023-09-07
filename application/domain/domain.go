package domain

import (
	"database/sql"
	_ "embed"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
	mwi18n "github.com/gobuffalo/mw-i18n/v2"
	"github.com/gofrs/uuid"
	"github.com/kelseyhightower/envconfig"

	"github.com/silinternational/cover-api/log"
)

//go:embed commit.txt
var Commit string

var AuthCallbackURL = Env.ApiBaseURL + "/auth/callback"

// T is the Buffalo i18n translator
var T *mwi18n.Translator

var extrasLock = sync.RWMutex{}

var AllowedFileUploadTypes = []string{
	"image/bmp",
	"image/gif",
	"image/jpeg",
	"image/png",
	"image/webp",
	"application/pdf",
	"text/plain",
	"text/plain; charset=utf-8",
	"text/csv",
}

// Context keys
const (
	ContextKeyCurrentUser = "current_user"
	ContextKeyExtras      = "extras"
	ContextKeyTx          = "tx"

	DefaultUIPath = "/home"

	EventPayloadID = "id"

	ExtrasIP     = "IP"
	ExtrasKey    = "key"
	ExtrasMethod = "method"
	ExtrasStatus = "status"
	ExtrasURI    = "URI"

	TypeClaim           = "claims"
	TypeClaimItem       = "claim-items"
	TypeClaimFile       = "claim-files"
	TypeEntityCode      = "entity-codes"
	TypeFile            = "files"
	TypeItem            = "items"
	TypeLedgerReport    = "ledger-reports"
	TypePolicy          = "policies"
	TypePolicyDependent = "policy-dependents"
	TypePolicyMember    = "policy-members"
	TypeStrike          = "strikes"
	TypeUser            = "users"
)

const (
	CurrencyFactor = 100
	DateFormat     = "2006-01-02"

	DurationDay  = time.Duration(time.Hour * 24)
	DurationWeek = time.Duration(DurationDay * 7)

	LocalizedDate = "2 January 2006"

	// How many hours old an item can be until it's not allowed to be deleted
	ItemDeleteCutOffHours = 72

	MaxFileSize = 1024 * 1024 * 10 // 10 Megabytes
	Megabyte    = 1048576

	ContentCSV  = "text/csv"
	ContentJson = "application/json"
	ContentZip  = "application/zip"
)

// Event Kinds
const (
	EventApiUserCreated      = "api:user:created"
	EventApiItemSubmitted    = "api:item:submitted"
	EventApiItemRevision     = "api:item:revision"
	EventApiItemAutoApproved = "api:item:autoapproved"
	EventApiItemApproved     = "api:item:approved"
	EventApiItemDenied       = "api:item:denied"

	EventApiClaimReview1     = "api:claim:review1"
	EventApiClaimRevision    = "api:claim:revision"
	EventApiClaimPreapproved = "api:claim:preapproved"
	EventApiClaimReceipt     = "api:claim:receipt"
	EventApiClaimReview2     = "api:claim:review2"
	EventApiClaimReview3     = "api:claim:review3"
	EventApiClaimApproved    = "api:claim:approved"
	EventApiClaimDenied      = "api:claim:denied"

	EventApiNotificationCreated = "api:notification:created"

	EventApiPolicyUserInviteCreated = "api:policy:invite:created"
)

// redirect url for after logout
var LogoutRedirectURL = Env.UIURL + "/logged-out"

// EnvDevelopment is used for various debugging aids
const EnvDevelopment = "development"

// EnvStaging is for the staging environment
const EnvStaging = "staging"

// EnvTest is for automated tests, during which some things are disabled
const EnvTest = "test"

// Env Holds the values of environment variables
var Env = readEnv()

type EnvStruct struct {
	GoEnv                      string `ignored:"true"`
	ApiBaseURL                 string `required:"true" split_words:"true"`
	AccessTokenLifetimeSeconds int    `default:"1166400" split_words:"true"` // 13.5 days
	AppName                    string `default:"Cover" split_words:"true"`
	HstsMaxAge                 int    `default:"3600" split_words:"true"` // default = 1 hour
	LogLevel                   string `default:"debug" split_words:"true"`
	AppNameLong                string `default:"Cover by SIL" split_words:"true"`
	Port                       int    `default:"3000"`

	ListenerDelayMilliseconds int `default:"1000" split_words:"true"`
	ListenerMaxRetries        int `default:"10" split_words:"true"`

	SessionSecret string `required:"true" split_words:"true"`
	UIURL         string `default:"http://missing.ui.url"`

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
	AwsAccessKeyID     string `split_words:"true"`
	AwsSecretAccessKey string `split_words:"true"`

	AwsS3Endpoint       string `split_words:"true"`
	AwsS3DisableSSL     bool   `split_words:"true"`
	AwsS3Bucket         string `split_words:"true"`
	AwsS3ACL            string `default:"private" split_words:"true"`
	AwsS3UrlLifeMinutes int    `default:"10" split_words:"true"`

	EmailFromAddress string `required:"true" split_words:"true"`
	EmailService     string `default:"ses" split_words:"true"`
	SupportEmail     string `default:"" split_words:"true"`
	SupportURL       string `default:"" split_words:"true"`
	FaqURL           string `default:"" split_words:"true"`

	InviteLifetimeDays int `default:"14" split_words:"true"`
	MaxFileDelete      int `default:"10" split_words:"true"`

	// The following will be multiplied by CurrencyFactor in readEnv()
	PolicyMaxCoverage       int `default:"50000" split_words:"true"`
	DependentAutoApproveMax int `default:"4000" split_words:"true"`
	PremiumMinimum          int `default:"25" split_words:"true"`

	// PremiumFactor is multiplied by CoverageAmount to calculate the annual premium of an item
	PremiumFactor           float64 `default:"0.02" split_words:"true"`
	RepairThreshold         float64 `default:"0.7" split_words:"true"`
	RepairThresholdString   string  `ignored:"true"`
	Deductible              float64 `default:"0.05"`
	DeductibleMinimumString string  `ignored:"true"`
	DeductibleIncrease      float64 `default:"0.2"` // Additional deductible per strike
	DeductibleMaximum       float64 `default:"0.45"`
	EvacuationDeductible    float64 `default:"0.333333333" split_words:"true"`
	StrikeLifetimeMonths    int     `default:"24" split_words:"true"`

	FiscalStartMonth   int    `default:"1" split_words:"true"`
	ExpenseAccount     string `required:"true" split_words:"true"`
	ClaimIncomeAccount string `required:"true" split_words:"true"`

	// For local development only, TLS can be disabled
	DisableTLS bool `default:"false" split_words:"true"`

	HouseholdIDLookupURL      string `default:"" split_words:"true"`
	HouseholdIDLookupUsername string `default:"" split_words:"true"`
	HouseholdIDLookupPassword string `default:"" split_words:"true"`

	UserWelcomeEmailIntro       string `default:"" split_words:"true"`
	UserWelcomeEmailPreviewText string `default:"" split_words:"true"`
	UserWelcomeEmailEnding      string `default:"" split_words:"true"`

	SandboxEmailAddress string `default:"" split_words:"true"`
}

func Init() {
	log.ErrLogger.Init(
		log.UseCommit(strings.TrimSpace(Commit)),
		log.UseEnv(Env.GoEnv),
		log.UseLevel(Env.LogLevel),
		log.UsePretty(Env.GoEnv == EnvDevelopment),
		log.UseRemote(Env.GoEnv != EnvTest),
	)

	log.Infof("ENV_VAR = %s", os.Getenv("ENV_VAR"))
	log.Infof("TEST_ENV_VAR = %s", os.Getenv("TEST_ENV_VAR"))
	log.Infof("ENV_ENCRYPTED_VAR = %s", os.Getenv("ENV_ENCRYPTED_VAR"))
}

// readEnv loads environment data into `Env`
func readEnv() *EnvStruct {
	env := &EnvStruct{}

	err := envconfig.Process("", env)
	if err != nil {
		panic("error loading env vars: " + err.Error())
	}

	checkSamlConfig(env)

	env.PolicyMaxCoverage *= CurrencyFactor
	env.DependentAutoApproveMax *= CurrencyFactor
	env.PremiumMinimum *= CurrencyFactor
	env.RepairThresholdString = fmt.Sprintf("%.2g%%", env.RepairThreshold*100)
	env.DeductibleMinimumString = fmt.Sprintf("%.2g%%", env.Deductible*100)

	//  Set an arbitrary but reasonable minimum lifetime for policy strikes
	if env.StrikeLifetimeMonths < 2 {
		env.StrikeLifetimeMonths = 2
	}

	// Doing this separately to avoid needing two environment variables for the same thing
	env.GoEnv = envy.Get("GO_ENV", EnvDevelopment)

	return env
}

// NewExtra Sets a new key-value pair in the `extras` entry of the context
func NewExtra(c buffalo.Context, key string, e any) {
	extras := getExtras(c)

	extrasLock.Lock()
	defer extrasLock.Unlock()
	extras[key] = e

	c.Set(ContextKeyExtras, extras)
}

func getExtras(c buffalo.Context) map[string]any {
	extras, _ := c.Value(ContextKeyExtras).(map[string]any)
	if extras == nil {
		extras = map[string]any{}
	}

	return extras
}

// GetUUID creates a new, unique version 4 (random) UUID and returns it
// as a uuid.UUID. Errors are ignored.
func GetUUID() uuid.UUID {
	id, err := uuid.NewV4()
	if err != nil {
		log.Error("error creating new uuid,", err)
	}
	return id
}

// EmailFromAddress combines a name with the configured from address for use in an email From header. If name is nil,
// only the App Name will be used.
func EmailFromAddress(name *string) string {
	addr := Env.AppNameLong + " <" + Env.EmailFromAddress + ">"
	if name != nil {
		return *name + " via " + addr
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
// were no rows in the result set for a sql query.
func IsOtherThanNoRows(err error) bool {
	if err == nil {
		return false
	}

	if strings.Contains(err.Error(), sql.ErrNoRows.Error()) {
		return false
	}

	return true
}

func MergeExtras(extras []map[string]any) map[string]any {
	allExtras := map[string]any{}

	// I didn't think I would need this, but without it at least one test was failing
	// The code allowed a map[string]any to get through (i.e. not in a slice)
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
	rand.Seed(time.Now().UnixNano())
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

// RandomInsecureIntInRange is insecure because it only uses the math/rand package
// and not the crypto/rand package
func RandomInsecureIntInRange(min, max int) int {
	if min >= max {
		panic("invalid parameters to RandomInsecureIntInRange: max of range must be greater than min.")
	}
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min // #nosec G404
}

// CalculatePartialYearValue returns the value multiplied by the number
// of days left between the startDate and the last day of the same year (inclusive)
// divided by 365  (rounded down)
// If the startDate is January 1, then the input value is returned.
// Note that the startDate's time of day is ignored.
func CalculatePartialYearValue(value int, startDate time.Time) int {
	if startDate.Month() == 1 && startDate.Day() == 1 {
		return value
	}
	startMidnight := startDate.Truncate(time.Hour * 24)
	thisYear := startDate.Year()
	newYears := time.Date(thisYear+1, 1, 1, 0, 0, 0, 0, time.UTC)
	hoursSince := int(newYears.Sub(startMidnight).Hours())

	daysSince := hoursSince / 24

	days := 365
	if IsLeapYear(startDate) {
		days = 366
	}

	return int(math.Round(float64(value*daysSince) / float64(days)))
}

// CalculateMonthlyRefundValue returns the value multiplied by the number
// of full calendar months left between the startDate and the end of
// the same year divided by 12  (rounded)
func CalculateMonthlyRefundValue(value int, startDate time.Time) int {
	remainingMonths := 12 - int(startDate.Month())
	return int(math.Round(float64(value*remainingMonths) / 12.0))
}

func BeginningOfDay(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func EndOfMonth(date time.Time) time.Time {
	return date.AddDate(0, 1, -date.Day())
}

// Returns a float as "dd%" (rounded and with no decimal places)
// Note: this won't look right if the input is greater than 1
func PercentString(d float64) string {
	return fmt.Sprintf("%.2g%%", d*100)
}

func IsLeapYear(t time.Time) bool {
	tt := time.Date(t.Year(), 2, 29, 0, 0, 0, 0, time.UTC)
	return tt.Day() == 29
}

func TimeBetween(t1, t2 time.Time) string {
	t1 = t1.Truncate(time.Minute)
	t2 = t2.Truncate(time.Minute)

	if t1 == t2 {
		return "just now"
	}

	var diff time.Duration
	if t1.Before(t2) {
		diff = t2.Sub(t1)
	} else {
		diff = t1.Sub(t2)
	}

	var unit string
	var n int

	if diff < time.Hour {
		n = int(diff / time.Minute)
		unit = "minute"
	} else if diff < DurationDay {
		n = int(diff / time.Hour)
		unit = "hour"
	} else {
		n = int(diff / DurationDay)
		unit = "day"
	}

	if n > 1 {
		unit += "s"
	}

	return fmt.Sprintf("%d %s ago", n, unit)
}

func IsProduction() bool {
	if strings.HasPrefix(Env.GoEnv, "prod") {
		return true
	}
	return false
}

func checkSamlConfig(env *EnvStruct) {
	if env.GoEnv == EnvDevelopment || env.GoEnv == EnvTest {
		return
	}
	if env.SamlIdpEntityId == "" {
		panic("required SAML variable SamlIdpEntityId is undefined")
	}
	if env.SamlSpEntityId == "" {
		panic("required SAML variable SamlSpEntityId is undefined")
	}
	if env.SamlSsoURL == "" {
		panic("required SAML variable SamlSsoURL is undefined")
	}
	if env.SamlSloURL == "" {
		panic("required SAML variable SamlSloURL is undefined")
	}
	if env.SamlAudienceUri == "" {
		panic("required SAML variable SamlAudienceUri is undefined")
	}
	if env.SamlAssertionConsumerServiceUrl == "" {
		panic("required SAML variable SamlAssertionConsumerServiceUrl is undefined")
	}
	if env.SamlIdpCert == "" {
		panic("required SAML variable SamlIdpCert is undefined")
	}
	if env.SamlSpCert == "" {
		panic("required SAML variable SamlSpCert is undefined")
	}
	if env.SamlSpPrivateKey == "" {
		panic("required SAML variable SamlSpPrivateKey is undefined")
	}
}
