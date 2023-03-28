package log

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// ErrLogger is an instance of ErrLogProxy, which can be used without access to the Buffalo context.
var ErrLogger ErrLogProxy

// ErrLogProxy wraps a logrus logger with optional hooks for sending to remote loggers like Rollbar
type ErrLogProxy struct {
	// LocalLog is sent to stdout
	LocalLog *logrus.Logger

	// RollbarHook is for sending entries to Rollbar
	rollbar *RollbarHook

	// SentryHook is for sending entries to Sentry
	sentry *SentryHook

	config Option
}

type Option struct {
	// commit is the git commit hash
	commit string

	// env is the operating environment, e.g. development, staging, production
	env string

	// level is the lowest level that will be sent to the remote log
	level string

	// pretty logs messages in color and with a timestamp, otherwise as json
	pretty bool

	// remote enables remote logging
	remote bool
}

func UseCommit(commitHash string) func(*Option) {
	return func(o *Option) { o.commit = commitHash }
}

func UseEnv(env string) func(*Option) {
	return func(o *Option) { o.env = env }
}

func UseLevel(level string) func(*Option) {
	return func(o *Option) { o.level = level }
}

func UsePretty(pretty bool) func(*Option) {
	return func(o *Option) { o.pretty = pretty }
}

func UseRemote(remote bool) func(*Option) {
	return func(o *Option) { o.remote = remote }
}

// Init does initial setup
func (e *ErrLogProxy) Init(options ...func(*Option)) {
	for _, option := range options {
		option(&e.config)
	}

	e.LocalLog = newLogrusLogger(e.config.level, e.config.pretty)

	if e.config.remote {
		e.rollbar = NewRollbarHook(e.config.env, e.config.commit)
		e.LocalLog.AddHook(e.rollbar)

		e.sentry = NewSentryHook(e.config.env, e.config.commit)
		e.LocalLog.AddHook(e.sentry)
	}
}

// newLogrusLogger creates a new logrus logger using TextFormatter for development and JSONFormatter otherwise
func newLogrusLogger(level string, pretty bool) *logrus.Logger {
	l := logrus.New()
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.DebugLevel
	}
	l.Level = logLevel
	l.SetOutput(os.Stdout)

	if pretty {
		l.Formatter = &logrus.TextFormatter{
			ForceColors:   true,
			FullTimestamp: true,
		}
	} else {
		l.Formatter = &logrus.JSONFormatter{
			DisableTimestamp: true,
		}
	}

	return l
}

func (e *ErrLogProxy) SetOutput(w io.Writer) {
	e.LocalLog.SetOutput(w)
}

func Panicf(format string, args ...any) {
	ErrLogger.LocalLog.Panicf(format, args...)
}

func Fatalf(format string, args ...any) {
	ErrLogger.LocalLog.Fatalf(format, args...)
}

func Errorf(format string, args ...any) {
	ErrLogger.LocalLog.Errorf(format, args...)
}

func Warningf(format string, args ...any) {
	ErrLogger.LocalLog.Warningf(format, args...)
}

func Infof(format string, args ...any) {
	ErrLogger.LocalLog.Infof(format, args...)
}

func Debugf(format string, args ...any) {
	ErrLogger.LocalLog.Debugf(format, args...)
}

func Tracef(format string, args ...any) {
	ErrLogger.LocalLog.Tracef(format, args...)
}

func Panic(args ...any) {
	ErrLogger.LocalLog.Panic(args...)
}

func Fatal(args ...any) {
	ErrLogger.LocalLog.Fatal(args...)
}

func Error(args ...any) {
	ErrLogger.LocalLog.Error(args...)
}

func Warning(args ...any) {
	ErrLogger.LocalLog.Warning(args...)
}

func Info(args ...any) {
	ErrLogger.LocalLog.Info(args...)
}

func Debug(args ...any) {
	ErrLogger.LocalLog.Debug(args...)
}

func Trace(args ...any) {
	ErrLogger.LocalLog.Trace(args...)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return ErrLogger.LocalLog.WithFields(fields)
}

func WithContext(ctx context.Context) *logrus.Entry {
	return ErrLogger.LocalLog.WithContext(ctx)
}

func SetUser(id, username, email string) {
	if !ErrLogger.config.remote {
		return
	}
	ErrLogger.rollbar.SetUser(id, username, email)
	ErrLogger.sentry.SetUser(id, username, email)
}
