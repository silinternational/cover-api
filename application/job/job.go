package job

import (
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/worker"
	"github.com/gobuffalo/pop/v5"
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

	user := models.GetDefaultSteward(models.DB)
	ctx.Set(domain.ContextKeyCurrentUser, user)

	if domain.Env.RollbarToken == "" || domain.Env.GoEnv == "test" {
		return ctx
	}

	client := rollbar.New(
		domain.Env.RollbarToken,
		domain.Env.GoEnv,
		"",
		"",
		domain.Env.RollbarServerRoot)
	defer func() {
		if err := client.Close(); err != nil {
			domain.ErrLogger.Printf("rollbar client.Close error: %s", err)
		}
	}()

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
func inactivateItemsHandler(_ worker.Args) error {
	defer resubmitInactivateJob()

	ctx := createJobContext()

	domain.Logger.Printf("starting inactivateItems job")
	nw := time.Now().UTC()

	err := models.DB.Transaction(func(tx *pop.Connection) error {
		ctx.Set(domain.ContextKeyTx, tx)
		var items models.Items
		return items.InactivateApprovedButEnded(ctx)
	})
	if err != nil {
		return err
	}

	domain.Logger.Printf("completed inactivateItems job in %v seconds", time.Since(nw).Seconds())
	return nil
}

func resubmitInactivateJob() {
	// Run twice a day, in case it errors out
	delay := time.Hour * 12

	// uncomment this in development, if you want it to run more often for debugging
	// delay = time.Duration(time.Second * 10)

	if err := SubmitDelayed(InactivateItems, delay, map[string]interface{}{}); err != nil {
		domain.ErrLogger.Printf("error resubmitting inactivateItemsHandler: " + err.Error())
	}
	return
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
