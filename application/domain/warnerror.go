package domain

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/gobuffalo/buffalo"
	"github.com/rollbar/rollbar-go"
)

// LogErrorMessage logs a message and sends it to Rollbar
func LogErrorMessage(c buffalo.Context, msg string, level string, extras ...map[string]any) {
	// Avoid panics running tests when c doesn't have the necessary nested methods
	logger := c.Logger()
	if logger == nil {
		return
	}

	es := MergeExtras(extras)
	if es == nil {
		es = map[string]any{}
	}

	rollbarMessage(c, level, msg, es)

	logger.Error(encodeLogMsg(msg, es))
}

func jsonMin(i any) ([]byte, error) {
	return json.Marshal(i)
}

func jsonIndented(i any) ([]byte, error) {
	return json.MarshalIndent(i, "", "  ")
}

// encodeLogMsg adds the message as an "extra" and returns a json-encoded copy of the resulting extras map
func encodeLogMsg(msg string, extras map[string]any) string {
	encoder := jsonMin
	if Env.GoEnv == EnvDevelopment {
		encoder = jsonIndented
	}

	if extras == nil {
		extras = map[string]any{}
	}
	extras["message"] = msg

	j, err := encoder(&extras)
	if err != nil {
		return "failed to json encode error message: " + err.Error()
	}
	return string(j)
}

// rollbarMessage is a wrapper function to call rollbar's client.MessageWithExtras function from client stored in context
func rollbarMessage(c buffalo.Context, level string, msg string, extras map[string]any) {
	rc, ok := c.Value(ContextKeyRollbar).(*rollbar.Client)
	if ok {
		rc.MessageWithExtras(level, msg, extras)
	}
}

// GetFunctionName provides the filename, line number, and function name of the caller, skipping the top `skip`
// functions on the stack.
func GetFunctionName(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "?"
	}

	fn := runtime.FuncForPC(pc)
	return fmt.Sprintf("%s:%d %s", file, line, fn.Name())
}
