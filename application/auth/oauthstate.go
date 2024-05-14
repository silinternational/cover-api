package auth

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"

	"github.com/silinternational/cover-api/domain"
)

// store can/should be set by applications using an oauth2 provider like google.
// The default is a cookie store.
// Borrowed from gothic
var (
	store        sessions.Store
	defaultStore sessions.Store
)

var keySet = false

func Init() {
	key := []byte(domain.Env.SessionSecret)
	keySet = len(key) != 0

	cookieStore := sessions.NewCookieStore([]byte(key))
	cookieStore.Options.HttpOnly = true
	store = cookieStore
	defaultStore = store
}

// sessionName is the key used to access the session store.
const sessionName = "_oauth2_session"

func CheckSessionStore() string {
	if !keySet && defaultStore == store {
		return "no SESSION_SECRET environment variable is set. " +
			"The default cookie store is not available and any calls will fail. " +
			"Ignore this warning if you are using a different store."
	}
	return ""
}

// ValidateState ensures that the state token param from the original
// AuthURL matches the one included in the current (callback) request.
func ValidateState(req *http.Request, sess goth.Session) error {
	rawAuthURL, err := sess.GetAuthURL()
	if err != nil {
		return err
	}

	authURL, err := url.Parse(rawAuthURL)
	if err != nil {
		return err
	}

	originalState := authURL.Query().Get("state")
	if originalState != "" && (originalState != req.URL.Query().Get("state")) {
		return errors.New("state token mismatch")
	}
	return nil
}

// StoreInSession stores a specified key/value pair in the session.
// Borrowed from gothic
func StoreInSession(key string, value string, req *http.Request, res http.ResponseWriter) error {
	session, _ := store.New(req, sessionName)

	if err := updateSessionValue(session, key, value); err != nil {
		return err
	}

	return session.Save(req, res)
}

// GetFromSession retrieves a previously-stored value from the session.
// If no value has previously been stored at the specified key, it will return an error.
// Borrowed from gothic
func GetFromSession(key string, req *http.Request) (string, error) {
	session, _ := store.Get(req, sessionName)
	value, err := getSessionValue(session, key)
	if err != nil {
		return "", errors.New("could not find a matching session for this request")
	}

	return value, nil
}

func getSessionValue(session *sessions.Session, key string) (string, error) {
	value := session.Values[key]
	if value == nil {
		return "", errors.New("could not find a matching session for this request")
	}

	rdata := strings.NewReader(value.(string))
	r, err := gzip.NewReader(rdata)
	if err != nil {
		return "", err
	}
	s, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return string(s), nil
}

func updateSessionValue(session *sessions.Session, key, value string) error {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(value)); err != nil {
		return err
	}
	if err := gz.Flush(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}

	session.Values[key] = b.String()
	return nil
}

// SetState sets the state string associated with the given request.
// If no state string is associated with the request, one will be generated.
// This state is sent to the provider and can be retrieved during the
// callback.
// Borrowed from gothic
var SetState = func(req *http.Request) string {
	state := req.URL.Query().Get("state")
	if len(state) > 0 {
		return state
	}

	// If a state query param is not passed in, generate a random
	// base64-encoded nonce so that the state on the auth URL
	// is unguessable, preventing CSRF attacks, as described in
	//
	// https://auth0.com/docs/protocols/oauth2/oauth-state#keep-reading
	nonceBytes := make([]byte, 64)
	_, err := io.ReadFull(rand.Reader, nonceBytes)
	if err != nil {
		panic("google_provider: source of randomness unavailable: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(nonceBytes)
}

// Logout invalidates a user session.
// Borrowed from gothic
func Logout(res http.ResponseWriter, req *http.Request) error {
	session, err := store.Get(req, sessionName)
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	session.Values = make(map[any]any)
	err = session.Save(req, res)
	if err != nil {
		return errors.New("Could not delete user session ")
	}
	return nil
}
