package job

import (
	"os"
	"runtime/debug"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/worker"
	"github.com/gobuffalo/pop/v6"
	"github.com/rollbar/rollbar-go"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

const (
	handlerKey = "job_handler"
	argJobType = "job_type"
)

const (
	InactivateItems = "inactivate_items"
)

var w *worker.Worker

var handlers = map[string]func(worker.Args) error{
	InactivateItems: inactivateItemsHandler,
}

// jobBuffaloContext is a buffalo context for jobs
type jobBuffaloContext struct {
	buffalo.DefaultContext
	params map[any]any
}

// Value returns the value associated with the given key in the context
func (j *jobBuffaloContext) Value(key any) any {
	return j.params[key]
}

// Set sets the value to be associated with the given key in the context. CAUTION: this is not thread-safe
func (j *jobBuffaloContext) Set(key string, val any) {
	j.params[key] = val
}

// createJobContext creates an empty context
func createJobContext() buffalo.Context {
	ctx := &jobBuffaloContext{
		params: map[any]any{},
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

func Init(appWorker *worker.Worker) {
	w = appWorker
	if err := (*w).Register(handlerKey, mainHandler); err != nil {
		domain.ErrLogger.Printf("error registering '%s' handler, %s", handlerKey, err)
	}

	delay := time.Second * 10

	// Kick off first run of inactivating items between 1h11 and 3h27 from now
	if domain.Env.GoEnv != domain.EnvDevelopment {
		randMins := time.Duration(domain.RandomInsecureIntInRange(71, 387))
		delay = randMins * time.Minute
	}

	if err := SubmitDelayed(InactivateItems, delay, map[string]any{}); err != nil {
		domain.ErrLogger.Printf("error initializing InactivateItems job: " + err.Error())
		os.Exit(1)
	}
}

func mainHandler(args worker.Args) error {
	jobType := args[argJobType].(string)

	defer func() {
		if err := recover(); err != nil {
			domain.ErrLogger.Printf("panic in job handler %s: %s\n%s", jobType, err, debug.Stack())
		}
	}()

	if err := handlers[jobType](args); err != nil {
		domain.ErrLogger.Printf("batch job %s failed: %s", jobType, err)
	}

	return nil
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
	// delay = time.Second * 10

	if err := SubmitDelayed(InactivateItems, delay, map[string]any{}); err != nil {
		domain.ErrLogger.Printf("error resubmitting inactivateItemsHandler: " + err.Error())
	}
	return
}

// SubmitDelayed enqueues a new Worker job for the given handler. Arguments can be provided in `args`.
func SubmitDelayed(jobType string, delay time.Duration, args map[string]any) error {
	job := worker.Job{
		Queue:   "default",
		Args:    args,
		Handler: handlerKey,
	}
	job.Args[argJobType] = jobType
	return (*w).PerformIn(job, delay)
}
