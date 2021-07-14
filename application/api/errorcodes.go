package api

const (
	CategoryDatabase  = ErrorCategory("DB")
	CategoryUser      = ErrorCategory("User") // used for errors related to user input, validation, etc.
	CategoryForbidden = ErrorCategory("Forbidden")
	CategoryNotFound  = ErrorCategory("NotFound")
	CategoryInternal  = ErrorCategory("Internal") // used for internal server errors, not related to bad user input
)

const (
	// General

	ErrorCreateFailure            = ErrorKey("ErrorCreateFailure")
	ErrorGenericInternalServer    = ErrorKey("ErrorGenericInternalServer")
	ErrorFailedToConvertToAPIType = ErrorKey("ErrorFailedToConvertToAPIType")
	ErrorInvalidRequestBody       = ErrorKey("ErrorInvalidRequestBody")
	ErrorMustBeAValidUUID         = ErrorKey("ErrorMustBeAValidUUID")
	ErrorNoRows                   = ErrorKey("ErrorNoRows")
	ErrorNotAuthorized            = ErrorKey("ErrorNotAuthorized")
	ErrorQueryFailure             = ErrorKey("ErrorQueryFailure")
	ErrorSaveFailure              = ErrorKey("ErrorSaveFailure")
	ErrorTransactionNotFound      = ErrorKey("ErrorTransactionNotFound")
	ErrorUnknown                  = ErrorKey("ErrorUnknown")
	ErrorUpdateFailure            = ErrorKey("ErrorUpdateFailure")
	ErrorValidation               = ErrorKey("ErrorValidation")

	// HTTP codes for customErrorHandler

	ErrorBadRequest           = ErrorKey("ErrorBadRequest")
	ErrorInternalServerError  = ErrorKey("ErrorInternalServerError")
	ErrorMethodNotAllowed     = ErrorKey("ErrorMethodNotAllowed")
	ErrorNotAuthenticated     = ErrorKey("ErrorNotAuthenticated")
	ErrorRouteNotFound        = ErrorKey("ErrorRouteNotFound")
	ErrorUnexpectedHTTPStatus = ErrorKey("ErrorUnexpectedHTTPStatus")
	ErrorUnprocessableEntity  = ErrorKey("ErrorUnprocessableEntity")

	// Authentication

	ErrorFindingUserByEmail             = ErrorKey("ErrorFindingUserByEmail")

)
