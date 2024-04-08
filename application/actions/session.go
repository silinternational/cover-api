package actions

import (
	"fmt"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

const sessionName = "session"

func sessionSetValue(c echo.Context, key, value interface{}) error {
	sess, err := getSession(c)
	if err != nil {
		return err
	}
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	sess.Values[key] = value
	if err = sess.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("sessionSetValue error, %w", err)
	}
	return nil
}

func sessionGetString(c echo.Context, key interface{}) (string, error) {
	if i, err := sessionGetValue(c, key); err != nil {
		return "", err
	} else {
		return i.(string), nil
	}
}

func sessionGetValue(c echo.Context, key interface{}) (interface{}, error) {
	sess, err := getSession(c)
	if err != nil {
		return nil, fmt.Errorf("unable to get value from session: %w", err)
	}
	if v, ok := sess.Values[key]; !ok {
		return nil, fmt.Errorf("key '%s' not found in session", key)
	} else {
		return v, nil
	}
}

func getSession(c echo.Context) (*sessions.Session, error) {
	if sess, err := session.Get(sessionName, c); err != nil {
		return nil, fmt.Errorf("error getting session from context: %w", err)
	} else {
		return sess, nil
	}
}

func clearSession(c echo.Context) error {
	sess, err := getSession(c)
	if err != nil {
		return fmt.Errorf("unable to get session in clearSession: %w", err)
	}
	// Clear the current session
	for k := range sess.Values {
		delete(sess.Values, k)
	}
	return nil
}
