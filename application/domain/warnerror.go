package domain

import (
	"context"
	"encoding/json"

	"github.com/gobuffalo/buffalo"
	"github.com/rollbar/rollbar-go"
)

// Error log error and send to Rollbar
func Error(c buffalo.Context, msg string, extras ...map[string]interface{}) {
	// Avoid panics running tests when c doesn't have the necessary nested methods
	logger := c.Logger()
	if logger == nil {
		return
	}

	es := MergeExtras(extras)
	if es == nil {
		es = map[string]interface{}{}
	}

	rollbarMessage(c, rollbar.ERR, msg, es)

	es["message"] = msg

	encoder := jsonMin
	if Env.GoEnv == "development" {
		encoder = jsonIndented
	}

	j, err := encoder(&es)
	if err != nil {
		logger.Error("failed to json encode error message: %s", err)
		return
	}

	logger.Error(string(j))
}

// Warn sends a message to Rollbar and to the local logger, including
// any extras found in the context.
func Warn(ctx context.Context, msg string) {
	bc := getBuffaloContext(ctx)

	extras := getExtras(bc)
	extrasLock.RLock()
	defer extrasLock.RUnlock()

	rollbarMessage(bc, rollbar.WARN, msg, extras)

	logger := bc.Logger()
	if logger != nil {
		logger.Warn(encodeLogMsg(msg, extras))
	}
}

// Info sends a message to the local logger, including any extras found in the context.
func Info(ctx context.Context, msg string) {
	bc := getBuffaloContext(ctx)

	extras := getExtras(bc)
	extrasLock.RLock()
	defer extrasLock.RUnlock()

	logger := bc.Logger()
	if logger != nil {
		logger.Info(encodeLogMsg(msg, extras))
	}
}

func jsonMin(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}

func jsonIndented(i interface{}) ([]byte, error) {
	return json.MarshalIndent(i, "", "  ")
}

func encodeLogMsg(msg string, extras map[string]interface{}) string {
	encoder := jsonMin
	if Env.GoEnv == "development" {
		encoder = jsonIndented
	}

	if extras == nil {
		extras = map[string]interface{}{}
	}
	extras["message"] = msg

	j, err := encoder(&extras)
	if err != nil {
		return "failed to json encode error message: " + err.Error()
	}
	return string(j)
}

// rollbarMessage is a wrapper function to call rollbar's client.MessageWithExtras function from client stored in context
func rollbarMessage(c buffalo.Context, level string, msg string, extras map[string]interface{}) {
	rc, ok := c.Value(ContextKeyRollbar).(*rollbar.Client)
	if ok {
		rc.MessageWithExtras(level, msg, extras)
	}
}
