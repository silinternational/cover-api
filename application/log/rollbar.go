package log

import (
	"context"
	"os"

	"github.com/gobuffalo/buffalo"
	"github.com/rollbar/rollbar-go"
	"github.com/sirupsen/logrus"
)

const ContextKeyRollbar = "rollbar"

var mapRollbarToLogrusLevel = map[logrus.Level]string{
	logrus.PanicLevel: rollbar.CRIT,
	logrus.FatalLevel: rollbar.CRIT,
	logrus.ErrorLevel: rollbar.ERR,
	logrus.WarnLevel:  rollbar.WARN,
	logrus.InfoLevel:  rollbar.INFO,
	logrus.DebugLevel: rollbar.DEBUG,
	logrus.TraceLevel: rollbar.DEBUG,
}

type RollbarHook struct {
	client *rollbar.Client
}

func NewRollbarHook(env, commit string) *RollbarHook {
	token := os.Getenv("ROLLBAR_SERVER_TOKEN")
	if token == "" {
		return nil
	}
	return &RollbarHook{client: rollbar.New(
		token,
		env,
		commit,
		"",
		os.Getenv("ROLLBAR_SERVER_ROOT"),
	)}
}

func RollbarMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		if ErrLogger.rollbar == nil || ErrLogger.rollbar.client == nil {
			return next(c)
		}

		// Make a copy of the client for each request context
		client := rollbar.New(
			ErrLogger.rollbar.client.Token(),
			ErrLogger.rollbar.client.Environment(),
			ErrLogger.rollbar.client.CodeVersion(),
			ErrLogger.rollbar.client.ServerHost(),
			ErrLogger.rollbar.client.ServerRoot(),
		)
		defer client.Close()

		c.Set(ContextKeyRollbar, &client)

		return next(c)
	}
}

func (r *RollbarHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel, logrus.WarnLevel}
}

func (r *RollbarHook) Fire(entry *logrus.Entry) error {
	extras := entry.Data

	if extras["status"] == 401 || extras["status"] == 404 {
		return nil
	}

	if ctx, ok := entry.Context.(buffalo.Context); ok {
		client := r.getClient(ctx)
		if client != nil {
			client.RequestMessageWithExtras(mapRollbarToLogrusLevel[entry.Level], ctx.Request(), entry.Message, extras)
			return nil
		}
	}
	r.client.MessageWithExtras(mapRollbarToLogrusLevel[entry.Level], entry.Message, extras)
	return nil
}

func (r *RollbarHook) SetUser(ctx context.Context, id, username, email string) {
	if r == nil || r.client == nil {
		return
	}
	contextClient := r.getClient(ctx)
	if contextClient != nil {
		contextClient.SetPerson(id, username, email)
	}
}

func (r *RollbarHook) getClient(ctx context.Context) *rollbar.Client {
	if c, ok := ctx.Value(ContextKeyRollbar).(*rollbar.Client); ok {
		return c
	}
	return nil
}
