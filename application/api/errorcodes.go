package api

const (
	CategoryDatabase  = ErrorCategory("Database")
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

	// Authentication
	ErrorAuthProvidersCallback   = ErrorKey("ErrorAuthProvidersCallback")
	ErrorGettingAuthURL          = ErrorKey("ErrorGettingAuthURL")
	ErrorLoadingAuthProvider     = ErrorKey("ErrorLoadingAuthProvider")
	ErrorMissingAuthEmail        = ErrorKey("ErrorMissingAuthEmail")
	ErrorMissingClientID         = ErrorKey("ErrorMissingClientID")
	ErrorMissingSessionAuthEmail = ErrorKey("ErrorMissingSessionAuthEmail")
	ErrorMissingSessionClientID  = ErrorKey("ErrorMissingSessionClientID")
	ErrorFindingUserByEmail      = ErrorKey("ErrorFindingUserByEmail")
)
