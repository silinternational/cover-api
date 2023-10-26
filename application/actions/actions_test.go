package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/httptest"
	"github.com/gobuffalo/pop/v6"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var futureDate = time.Date(2222, 12, 31, 12, 59, 59, 59, time.UTC)

type ActionSuite struct {
	suite.Suite
	*require.Assertions
	app     *buffalo.App
	DB      *pop.Connection
	Session *buffalo.Session
}

// HTML creates an httptest.Request with HTML content type.
func (as *ActionSuite) HTML(u string, args ...any) *httptest.Request {
	return httptest.New(as.app).HTML(u, args...)
}

// JSON creates an httptest.JSON request
func (as *ActionSuite) JSON(u string, args ...any) *httptest.JSON {
	return httptest.New(as.app).JSON(u, args...)
}

func Test_ActionSuite(t *testing.T) {
	as := &ActionSuite{
		app: App(),
	}
	c, err := pop.Connect(domain.EnvTest)
	if err == nil {
		models.DB = c
		as.DB = c
	}
	suite.Run(t, as)
}

// SetupTest sets the test suite to abort on first failure and sets the session store
func (as *ActionSuite) SetupTest() {
	as.Assertions = require.New(as.T())

	as.app.SessionStore = newSessionStore()
	s, _ := as.app.SessionStore.New(nil, as.app.SessionName)
	as.Session = &buffalo.Session{
		Session: s,
	}

	models.DestroyAll()
	models.InsertTestData()
}

func (as *ActionSuite) verifyResponseData(wantData []string, body string, msg string) {
	var b bytes.Buffer
	as.NoError(json.Indent(&b, []byte(body), "", "    "))
	for _, w := range wantData {
		if !strings.Contains(body, w) {
			as.Fail(fmt.Sprintf("%s response data is not correct\nwanted: %s\nin body:\n%s\n", msg, w, b.String()))
		}
	}
}

// sessionStore copied from gobuffalo/suite session.go
type sessionStore struct {
	sessions map[string]*sessions.Session
}

func (s *sessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	if s, ok := s.sessions[name]; ok {
		return s, nil
	}
	return s.New(r, name)
}

func (s *sessionStore) New(r *http.Request, name string) (*sessions.Session, error) {
	sess := sessions.NewSession(s, name)
	s.sessions[name] = sess
	return sess, nil
}

func (s *sessionStore) Save(r *http.Request, w http.ResponseWriter, sess *sessions.Session) error {
	if s.sessions == nil {
		s.sessions = map[string]*sessions.Session{}
	}
	s.sessions[sess.Name()] = sess
	return nil
}

// NewSessionStore for action suite
func newSessionStore() sessions.Store {
	return &sessionStore{
		sessions: map[string]*sessions.Session{},
	}
}

func (as *ActionSuite) decodeBody(body []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}
