package domain

import (
	"context"
	"encoding/json"

	"github.com/gobuffalo/buffalo"
	"github.com/rollbar/rollbar-go"
)

// Error sends a message to Rollbar and to the local logger, including
// any extras found in the context.
func Error(ctx context.Context, msg string) {
	bc := getBuffaloContext(ctx)

	extras := getExtras(bc)
	extrasLock.RLock()
	defer extrasLock.RUnlock()

	rollbarMessage(bc, rollbar.ERR, msg, extras)

	logger := bc.Logger()
	if logger != nil {
		logger.Error(encodeLogMsg(msg, extras))
	}
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
		return
	}
}
