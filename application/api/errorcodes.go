package api

const (
	CategoryDatabase     = ErrorCategory("Database")
	CategoryUser         = ErrorCategory("User") // used for errors related to user input, validation, etc.
	CategoryForbidden    = ErrorCategory("Forbidden")
	CategoryUnauthorized = ErrorCategory("Unauthorized")
	CategoryNotFound     = ErrorCategory("NotFound")
	CategoryInternal     = ErrorCategory("Internal") // used for internal server errors, not related to bad user input
)

const (
	// General

	ErrorCreateFailure            = ErrorKey("ErrorCreateFailure")
	ErrorGenericInternalServer    = ErrorKey("ErrorGenericInternalServer")
	ErrorFailedToConvertToAPIType = ErrorKey("ErrorFailedToConvertToAPIType")
	ErrorInvalidRequestBody       = ErrorKey("ErrorInvalidRequestBody")
	ErrorMissingSessionKey        = ErrorKey("ErrorMissingSessionKey")
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
	ErrorAuthProvidersCallback  = ErrorKey("ErrorAuthProvidersCallback")
	ErrorAuthProvidersLogout    = ErrorKey("ErrorAuthProvidersLogout")
	ErrorCreatingAccessToken    = ErrorKey("ErrorCreatingAccessToken")
	ErrorDeletingAccessToken    = ErrorKey("ErrorDeletingAccessToken")
	ErrorFindingAccessToken     = ErrorKey("ErrorFindingAccessToken")
	ErrorGettingAuthURL         = ErrorKey("ErrorGettingAuthURL")
	ErrorLoadingAuthProvider    = ErrorKey("ErrorLoadingAuthProvider")
	ErrorMissingAuthEmail       = ErrorKey("ErrorMissingAuthEmail")
	ErrorMissingClientID        = ErrorKey("ErrorMissingClientID")
	ErrorMissingLogoutToken     = ErrorKey("ErrorMissingLogoutToken")
	ErrorMissingSessionClientID = ErrorKey("ErrorMissingSessionClientID")
	ErrorWithAuthUser           = ErrorKey("ErrorWithAuthUser")

	// Authorization
	ErrorInvalidResourceID = ErrorKey("ErrorInvalidResourceID")
	ErrorResourceNotFound  = ErrorKey("ErrorResourceNotFound")

	// Policy
	ErrorPolicyNotFound           = ErrorKey("ErrorPolicyNotFound")
	ErrorPolicyUpdateInvalidInput = ErrorKey("ErrorPolicyUpdateInvalidInput")

	// PolicyDependent
	ErrorPolicyDependentCreateInvalidInput = ErrorKey("ErrorPolicyDependentCreateInvalidInput")
	ErrorPolicyDependentCreate             = ErrorKey("ErrorPolicyDependentCreate")
)
