package job

import (
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/worker"
	"github.com/rollbar/rollbar-go"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

const (
	InactivateItems = "inactivate_items"
)

var w worker.Worker

// jobBuffaloContext is a buffalo context for jobs
type jobBuffaloContext struct {
	buffalo.DefaultContext
	params map[interface{}]interface{}
}

// Value returns the value associated with the given key in the test context
func (j *jobBuffaloContext) Value(key interface{}) interface{} {
	return j.params[key]
}

// Set sets the value to be associated with the given key in the test context
func (j *jobBuffaloContext) Set(key string, val interface{}) {
	j.params[key] = val
}

// createJobContext creates an empty context
func createJobContext() buffalo.Context {
	ctx := &jobBuffaloContext{
		params: map[interface{}]interface{}{},
	}

	if domain.Env.RollbarToken == "" || domain.Env.GoEnv == "test" {
		return ctx
	}

	client := rollbar.New(
		domain.Env.RollbarToken,
		domain.Env.GoEnv,
		"",
		"",
		domain.Env.RollbarServerRoot)
	defer client.Close()

	ctx.Set(domain.ContextKeyRollbar, client)
	return ctx
}

var handlers = map[string]func(worker.Args) error{
	InactivateItems: inactivateItemsHandler,
}

func init() {
	w = worker.NewSimple()
	for key, handler := range handlers {
		if err := w.Register(key, handler); err != nil {
			domain.ErrLogger.Printf("error registering '%s' handler, %s", key, err)
		}
	}
}

// inactivateItemsHandler is the Worker handler for inactivating items that
// have a coverage end date in the past
func inactivateItemsHandler(args worker.Args) error {
	defer resubmitInactivateJob()

	ctx := createJobContext()

	domain.Logger.Printf("starting inactivateItems job")
	nw := time.Now().UTC()

	var items models.Items
	if err := items.InactivateApprovedButEnded(ctx); err != nil {
		return err
	}

	domain.Logger.Printf("completed inactivateItems job in %v seconds", time.Since(nw).Seconds())
	return nil
}

func resubmitInactivateJob() error {
	// Run twice a day, in case it errors out
	delay := time.Duration(time.Hour * 12)

	// uncomment this in development, if you want it to run more often for debugging
	// delay = time.Duration(time.Second * 10)

	if err := SubmitDelayed(InactivateItems, delay, map[string]interface{}{}); err != nil {
		domain.ErrLogger.Printf("error resubmitting inactivateItemsHandler: " + err.Error())
	}
	return nil
}

// SubmitDelayed enqueues a new Worker job for the given handler. Arguments can be provided in `args`.
func SubmitDelayed(handler string, delay time.Duration, args map[string]interface{}) error {
	job := worker.Job{
		Queue:   "default",
		Args:    args,
		Handler: handler,
	}
	return w.PerformIn(job, delay)
}
