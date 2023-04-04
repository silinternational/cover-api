package log

import (
	"context"
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
	_ "github.com/getsentry/sentry-go"
	"github.com/gobuffalo/buffalo"
	"github.com/sirupsen/logrus"
)

const ContextKeySentryHub = "sentry_hub"

var mapLogrusToSentryLevel = map[logrus.Level]sentry.Level{
	logrus.PanicLevel: sentry.LevelFatal,
	logrus.FatalLevel: sentry.LevelFatal,
	logrus.ErrorLevel: sentry.LevelError,
	logrus.WarnLevel:  sentry.LevelWarning,
	logrus.InfoLevel:  sentry.LevelInfo,
	logrus.DebugLevel: sentry.LevelDebug,
	logrus.TraceLevel: sentry.LevelDebug,
}

type SentryHook struct {
	hub *sentry.Hub
}

func SentryMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		r := c.Request()
		hub := sentry.GetHubFromContext(r.Context())

		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}

		c.Set(ContextKeySentryHub, hub)
		return next(c)
	}
}

func (s *SentryHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel, logrus.WarnLevel}
}

func (s *SentryHook) Fire(entry *logrus.Entry) error {
	extras := entry.Data

	if extras["status"] == 401 || extras["status"] == 404 {
		return nil
	}

	event := sentry.Event{
		Extra:   extras,
		Level:   mapLogrusToSentryLevel[entry.Level],
		Message: entry.Message,
	}
	if c, ok := entry.Context.(buffalo.Context); ok {
		event.Request = sentry.NewRequest(c.Request())
	}

	sentry.CaptureEvent(&event)
	return nil
}

func (s *SentryHook) SetUser(id, username, email string) {
	if s == nil || s.hub == nil {
		return
	}

	s.hub.Scope().SetUser(sentry.User{
		ID:       id,
		Username: username,
		Email:    email,
	})
}

func NewSentryHook(env, commit string) *SentryHook {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return nil
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          commit,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		panic(fmt.Sprintf("sentry.Init: %s", err))
	}

	return &SentryHook{hub: sentry.CurrentHub()}
}
