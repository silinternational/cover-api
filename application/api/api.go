package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/domain"
)

const (
	ResourceSubmit     = "submit"
	ResourceRevision   = "revision"
	ResourcePreapprove = "preapprove"
	ResourceReceipt    = "receipt"
	ResourceApprove    = "approve"
	ResourceDeny       = "deny"
	ResourceRecent     = "recent"
	ResourceStrikes    = "strikes"
)

// swagger:model
type ListResponse struct {
	// Meta contains pagination data
	Meta Meta `json:"meta"`

	// Data containing the relevant list type
	Data interface{} `json:"data"`
}

// TODO: implement Meta type to provide pagination properties
type Meta struct {
	*pop.Paginator
}

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

// LoadTranslatedMessage assigns the error message by translating the Key into a user-friendly string, either
// from a list of translated strings (see errors.en) or by breaking down the Key into individual words
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

	if len(words) > 1 && words[0] == "Error" {
		words = words[1:]
	}

	count := len(words)
	newWords := []string{}

	// Lowercase all but first word.
	for i := 0; i < count; i++ {
		// If a word is longer than one character, just use it as is
		if len(words[i]) > 1 {
			newWords = append(newWords, strings.ToLower(words[i]))
			continue
		}

		// Combine single character words
		next := words[i]
		for j := i + 1; j < count; j++ {
			if len(words[j]) == 1 {
				next += words[j]
				i++ // avoid reprocessing the same word
			} else {
				break
			}
		}
		newWords = append(newWords, strings.ToLower(next))
	}

	firstUpper := strings.ToUpper(newWords[0][0:1])
	newWords[0] = firstUpper + newWords[0][1:]

	return strings.Join(newWords, " ")
}

// Currency is in US Dollars, specified as an integer representing cents ($0.01 USD is represented as 1 and $105.36 as 10536)
// swagger:model
type Currency int

func (c Currency) String() string {
	return fmt.Sprintf("%0.2f", float32(c)/domain.CurrencyFactor)
}

// swagger:model
type RecentObjects struct {
	Items  RecentItems
	Claims RecentClaims
}
