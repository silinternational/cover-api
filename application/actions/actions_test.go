package actions

import (
	"net/http"
	"testing"
	"time"

	"github.com/silinternational/riskman-api/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/httptest"
	"github.com/gobuffalo/pop/v5"
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
func (as *ActionSuite) HTML(u string, args ...interface{}) *httptest.Request {
	return httptest.New(as.app).HTML(u, args...)
}

// JSON creates an httptest.JSON request
func (as *ActionSuite) JSON(u string, args ...interface{}) *httptest.JSON {
	return httptest.New(as.app).JSON(u, args...)
}

func Test_ActionSuite(t *testing.T) {
	as := &ActionSuite{
		app: App(),
	}
	c, err := pop.Connect("test")
	if err == nil {
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
