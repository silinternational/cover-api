package job

import (
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/worker"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

const (
	InactivateItems = "inactivate_items"
)

var w worker.Worker

// TestBuffaloContext is a buffalo context user in tests
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

// CreateTestContext sets the domain.ContextKeyCurrentUser to the user param in the TestBuffaloContext
func createJobContext() buffalo.Context {
	ctx := &jobBuffaloContext{
		params: map[interface{}]interface{}{},
	}
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

	var items models.Items
	return items.InactivateActiveButEnded(ctx)
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
