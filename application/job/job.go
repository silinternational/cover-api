package job

import (
	"os"
	"runtime/debug"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/worker"
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
	AnnualRenewal   = "annual_renewal"
)

var w *worker.Worker

var handlers = map[string]func(worker.Args) error{
	InactivateItems: inactivateItemsHandler,
	AnnualRenewal:   annualRenewalHandler,
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

	domain.Logger.Printf("starting %s job", jobType)
	start := time.Now().UTC()

	defer func() {
		if err := recover(); err != nil {
			domain.ErrLogger.Printf("panic in job handler %s: %s\n%s", jobType, err, debug.Stack())
		}
	}()

	if err := handlers[jobType](args); err != nil {
		domain.ErrLogger.Printf("batch job %s failed: %s", jobType, err)
	}

	domain.Logger.Printf("completed %s job in %s seconds", jobType, time.Since(start))
	return nil
}

// Submit enqueues a new Worker job for the given job type. Arguments can be provided in `args`.
func Submit(jobType string, args map[string]any) error {
	if domain.Env.GoEnv == domain.EnvTest {
		return nil
	}
	job := worker.Job{
		Queue:   "default",
		Args:    args,
		Handler: handlerKey,
	}
	job.Args[argJobType] = jobType
	return (*w).Perform(job)
}

// SubmitDelayed enqueues a delayed Worker job for the given job type. Arguments can be provided in `args`.
func SubmitDelayed(jobType string, delay time.Duration, args map[string]any) error {
	if domain.Env.GoEnv == domain.EnvTest {
		return nil
	}
	job := worker.Job{
		Queue:   "default",
		Args:    args,
		Handler: handlerKey,
	}
	job.Args[argJobType] = jobType
	return (*w).PerformIn(job, delay)
}
