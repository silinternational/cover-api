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

	// File
	ErrorReceivingFile           = ErrorKey("ErrorReceivingFile")
	ErrorStoreFileBadContentType = ErrorKey("ErrorStoreFileBadContentType")
	ErrorStoreFileTooLarge       = ErrorKey("ErrorStoreFileTooLarge")
	ErrorUnableToReadFile        = ErrorKey("ErrorUnableToReadFile")
	ErrorUnableToStoreFile       = ErrorKey("ErrorUnableToStoreFile")

	// Claim
	ErrorClaimFromContext      = ErrorKey("ErrorClaimFromContext")
	ErrorClaimStatus           = ErrorKey("ErrorClaimStatus")
	ErrorClaimMissingClaimItem = ErrorKey("ErrorClaimMissingClaimItem")

	// Item
	ErrorItemFromContext              = ErrorKey("ErrorItemFromContext")
	ErrorItemInvalidPurchaseDate      = ErrorKey("ErrorItemInvalidPurchaseDate")
	ErrorItemCoverageAmount           = ErrorKey("ErrorItemCoverageAmount")
	ErrorItemInvalidCoverageStartDate = ErrorKey("ErrorItemInvalidCoverageStartDate")
	ErrorInvalidCategory              = ErrorKey("ErrorInvalidCategory")

	// Policy
	ErrorPolicyFromContext        = ErrorKey("ErrorPolicyFromContext")
	ErrorPolicyNotFound           = ErrorKey("ErrorPolicyNotFound")
	ErrorPolicyLoadingItems       = ErrorKey("ErrorPolicyLoadingItems")
	ErrorPolicyUpdateInvalidInput = ErrorKey("ErrorPolicyUpdateInvalidInput")

	// PolicyDependent
	ErrorPolicyDependentCreate = ErrorKey("ErrorPolicyDependentCreate")

	// Claim
	ErrorClaimItemCreateInvalidInput = ErrorKey("ErrorClaimItemCreateInvalidInput")
)
