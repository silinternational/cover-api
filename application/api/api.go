package api

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/domain"
)

const (
	ResourceSubmit     = "submit"
	ResourceRevision   = "revision"
	ResourcePreapprove = "preapprove"
	ResourceReceipt    = "receipt"
	ResourceApprove    = "approve"
	ResourceDeny       = "deny"
)

type ErrorKey string

func (e ErrorKey) String() string {
	return string(e)
}

type ErrorCategory string

func (e ErrorCategory) String() string {
	return string(e)
}

// AppError holds information that is helpful in logging and reporting api errors
type AppError struct {
	Err error `json:"-"`

	// Don't change the value of these Key entries without making a corresponding change on the UI,
	// since these will be converted to human-friendly texts for presentation to the user
	Key ErrorKey `json:"key"`

	HttpStatus int `json:"status"`

	// detailed error message for debugging
	DebugMsg string `json:"debug_msg,omitempty"`

	Category ErrorCategory `json:"-"`

	Message string `json:"message"`

	// Extra data providing detail about the error condition, only provided in development mode
	Extras map[string]interface{} `json:"extras,omitempty"`

	// URL to redirect, if HttpStatus is in 300-series
	RedirectURL string `json:"-"`
}

func (a *AppError) Error() string {
	if a.Err == nil {
		return ""
	}
	return a.Err.Error()
}

func (a *AppError) Unwrap() error {
	return a.Err
}

// NewAppError returns a new AppError with its Err, Key and Category sset
func NewAppError(err error, key ErrorKey, category ErrorCategory) *AppError {
	return &AppError{
		Err:      err,
		Key:      key,
		Category: category,
	}
}

// SetHttpStatusFromCategory assigns the appropriate HTTP status based on the error category, if not
// already set.
func (a *AppError) SetHttpStatusFromCategory() {
	if a.HttpStatus != 0 {
		return
	}

	switch a.Category {
	case CategoryInternal, CategoryDatabase:
		a.HttpStatus = http.StatusInternalServerError
	case CategoryForbidden, CategoryNotFound:
		a.HttpStatus = http.StatusNotFound
	case CategoryUnauthorized:
		a.HttpStatus = http.StatusUnauthorized
	default:
		a.HttpStatus = http.StatusBadRequest
	}
}

// LoadTranslatedMessage assigns the error message by translating the Key into a user-friendly string, unless
// the HttpStatus is 500 in which case a standard message is used.
func (a *AppError) LoadTranslatedMessage(c buffalo.Context) {
	key := a.Key

	if a.HttpStatus == http.StatusInternalServerError {
		key = ErrorGenericInternalServer
	}

	msgID := "Error." + key.String()
	a.Message = domain.T.Translate(c, msgID, a.Extras)
	if a.Message == msgID {
		a.Message = keyToReadableString(a.Key.String())
	}
}

// keyToReadableString takes a key like ErrorSomethingSomethingOther and returns Error something something other
// although it will lose initial lowercase letters, if it has a non-initial uppercase letter
func keyToReadableString(key string) string {
	re := regexp.MustCompile(`[A-Z][^A-Z]*`)
	words := re.FindAllString(key, -1)

	if len(words) == 0 {
		return key
	}

	// Lowercase all but first word
	for i := 1; i < len(words); i++ {
		words[i] = strings.ToLower(words[i])
	}

	return strings.Join(words, " ")
}

// MergeExtras returns a single map with the all the key-values pairs of the input map
//  Key-value pairs in later maps will overwrite matching ones from earlier maps
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
