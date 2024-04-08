package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo-contrib/session"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var futureDate = time.Date(2222, 12, 31, 12, 59, 59, 59, time.UTC)

type ActionSuite struct {
	suite.Suite
	*require.Assertions
	app     *echo.Echo
	DB      *pop.Connection
	Session *buffalo.Session
}

func (as *ActionSuite) request(method, path, token string, input any) ([]byte, int) {
	var r io.Reader
	if input != nil {
		j, _ := json.Marshal(&input)
		r = bytes.NewReader(j)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)

	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)
	body, err := io.ReadAll(res.Body)
	as.NoError(err)
	return body, res.Code
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

	models.DestroyAll()
	models.InsertTestData()

	as.app.Use(session.Middleware(sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))))
}

func (as *ActionSuite) verifyResponseData(wantData []string, body []byte, msg string) {
	var b bytes.Buffer
	as.NoError(json.Indent(&b, body, "", "    "))
	for _, w := range wantData {
		if !strings.Contains(string(body), w) {
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

func (as *ActionSuite) Test_robots() {
	body, status := as.request("GET", "/robots.txt", "", nil)
	as.Equal(http.StatusOK, status, "incorrect status code returned: %d\n%s", status, body)
	as.True(strings.HasPrefix(string(body), "User-agent"),
		"incorrect response body:\n%s", body)
}
