# Contributing to Riskman-API

#### Table of Contents

[IDE configuration](#ide-configuration)

[Coding Style](#coding-style)


## IDE configuration

### Custom tags
Add "development" to the "Custom tags" setting in "Go - Build Tags & Vendoring" section of Goland Preferences. This ensures that the test fixture generation functions in `testutils.go` are not ignored by the IDE.

### Editorconfig

This project includes an .editorconfig file to enforce consistent formatting. See the [Editorconfig](https://editorconfig.org/) page for details. Enable this feature in your IDE to activate the configuration.

### Go formatting

Because Go has one code formatting standard, this project uses that
standard. To stay consistent, enable `goimports` in your editor or IDE to
format your code before it's committed. For example, in Goland, go to Settings -
Tools - File Watchers, add and enable `goimports`. Recommended, but not necessary: run [gofumpt](https://github.com/mvdan/gofumpt) as a file watcher to further format code in a consistent pattern.

## Coding Style

### Function naming

Within the `model` package, we have decided on function names starting with
certain standardized verbs: Get, Find, Create, Delete. When possible, functions
should have a model struct attached as a pointer: `func (r *Request)
FindByUUID(uuid string) error`.

### Unit test naming

Unit test functions that test methods should be named like
`TestObject_FunctionName` where `Object` is the name of the struct and
`FunctionName` is the name of the function under test.

### Test suites

Use Buffalo ([strechr/testify](https://github.com/stretchr/testify)) test
suites. If not all tests in a package that uses Buffalo suites use the correct
syntax, then running `buffalo test -m TestObject_FunctionName` will run the
expected test and any standard Go test functions/suites. For example, since the
`models` package has a `models_test` suite, all tests in this package should be
of the form:
```go
func (ms *ModelSuite) TestObject_FunctionName() {
}
```
rather than
```go
func Test_FunctionName(t *testing.T) {
}
```

### Running tests manually

To run all tests, run `make test`.

To run a single test:
1. run `make testenv` - this starts the test container and drops you into a bash prompt, from which you can run test commands.
2. `buffalo test actions -m "Test_Name"` will run any tests matching "Test_Name" in the "actions" package.
3. (alternative) `go test -v -tags development ./actions -testify.m "Test_Name"` - this runs more quickly than `buffalo test` and allows you to use go test flags like `-v`. The `-tags development` is applied by `buffalo test` but not by `go test` and is required in order to include the test fixture generation in `testutils.go`

### Database Queries

For simple queries and simple joins, Pop provides a good API based on
model struct annotations. These should be used where possible. Do not assume,
however, that objects passed from other functions are pre-populated with
data from related objects. If related data is required, call the `tx.Load`
function.

Complex queries and joins can be accomplished using the model fields and
iterating over the attached lists. This ends up being more complex and
difficult to read. We have determined it is better to use raw SQL in these
situations. For example:

```go
    var t Threads
    query := DB.Q().LeftJoin("thread_participants tp", "threads.id = tp.thread_id")
    query = query.Where("tp.user_id = ?", u.ID)
    if err := query.All(&t); err != nil {
        return nil, err
    }
```

### Error handling and presentation

#### REST API responses

Errors occurring in the processing of REST API requests should result in a 400-
or 500-level http response with a json body like:

```json
{
  "code": 400,
  "key": "ErrorKeyExample",
  "message": "This is an example error message"
}
``` 

The type `api.AppError` will render as required above by passing it to
`actions.reportError`. An `AppError` should be created by calling
`api.NewAppError` as deep into the call stack as needed to provide a detailed
key and specific category. If `actions.reportError` receives a generic `error`,
it will render with key `UnknownError` and HTTP status 500 and the error string
in the `DebugMsg`.

| Category          | HTTP Status |
|-------------------|-------------|
| CategoryInternal  | 500         |
| CategoryDatabase  | 500         |
| CategoryForbidden | 404         |
| CategoryNotFound  | 404         |
| CategoryUser      | 400         |

#### Internal error logging

Errors that do not justify an error being passed to the API client may be logged
to `stderr` and Rollbar using `domain.Error` if context is available, or
`domain.ErrLogger.printf` if no context is available.

`domain.Warn` can be used to log at level "warning" and also send to Rollbar

`domain.Info` or `domain.Logger.printf` will log but not send to Rollbar.

