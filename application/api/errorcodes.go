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
	ErrorDestroyFailure           = ErrorKey("ErrorDestroyFailure")
	ErrorGenericInternalServer    = ErrorKey("ErrorGenericInternalServer")
	ErrorFailedToConvertToAPIType = ErrorKey("ErrorFailedToConvertToAPIType")
	ErrorForeignKeyViolation      = ErrorKey("ErrorForeignKeyViolation")
	ErrorInvalidRequestBody       = ErrorKey("ErrorInvalidRequestBody")
	ErrorMissingSessionKey        = ErrorKey("ErrorMissingSessionKey")
	ErrorMustBeAValidUUID         = ErrorKey("ErrorMustBeAValidUUID")
	ErrorNoRows                   = ErrorKey("ErrorNoRows")
	ErrorNotAuthorized            = ErrorKey("ErrorNotAuthorized")
	ErrorQueryFailure             = ErrorKey("ErrorQueryFailure")
	ErrorSaveFailure              = ErrorKey("ErrorSaveFailure")
	ErrorTransactionNotFound      = ErrorKey("ErrorTransactionNotFound")
	ErrorUniqueKeyViolation       = ErrorKey("ErrorUniqueKeyViolation")
	ErrorUnknown                  = ErrorKey("ErrorUnknown")
	ErrorUpdateFailure            = ErrorKey("ErrorUpdateFailure")
	ErrorValidation               = ErrorKey("ErrorValidation")

	// Authentication
	ErrorAuthProvidersCallback    = ErrorKey("ErrorAuthProvidersCallback")
	ErrorAuthProvidersLogout      = ErrorKey("ErrorAuthProvidersLogout")
	ErrorCreatingAccessToken      = ErrorKey("ErrorCreatingAccessToken")
	ErrorDeletingAccessToken      = ErrorKey("ErrorDeletingAccessToken")
	ErrorFindingAccessToken       = ErrorKey("ErrorFindingAccessToken")
	ErrorGettingAuthURL           = ErrorKey("ErrorGettingAuthURL")
	ErrorInviteExpired            = ErrorKey("ErrorInviteExpired")
	ErrorLoadingAuthProvider      = ErrorKey("ErrorLoadingAuthProvider")
	ErrorMissingAuthEmail         = ErrorKey("ErrorMissingAuthEmail")
	ErrorMissingClientID          = ErrorKey("ErrorMissingClientID")
	ErrorMissingLogoutToken       = ErrorKey("ErrorMissingLogoutToken")
	ErrorMissingSessionClientID   = ErrorKey("ErrorMissingSessionClientID")
	ErrorProcessingAuthInviteCode = ErrorKey("ErrorProcessingAuthInviteCode")
	ErrorWithAuthUser             = ErrorKey("ErrorWithAuthUser")

	// Authorization
	ErrorInvalidResourceID = ErrorKey("ErrorInvalidResourceID")
	ErrorResourceNotFound  = ErrorKey("ErrorResourceNotFound")

	// File
	ErrorFileAlreadyLinked       = ErrorKey("ErrorFileAlreadyLinked")
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
	ErrorItemNullAccountablePerson    = ErrorKey("ErrorItemNullAccountablePerson")
	ErrorItemCoverageAmount           = ErrorKey("ErrorItemCoverageAmount")
	ErrorItemInvalidCoverageStartDate = ErrorKey("ErrorItemInvalidCoverageStartDate")
	ErrorItemInvalidCoverageEndDate   = ErrorKey("ErrorItemInvalidCoverageEndDate")
	ErrorInvalidCategory              = ErrorKey("ErrorInvalidCategory")
	ErrorItemHasActiveClaim           = ErrorKey("ErrorItemHasActiveClaim")

	// Ledger
	ErrorItemInvalidEndDate = ErrorKey("ErrorItemInvalidEndDate")
	ErrorInvalidReportType  = ErrorKey("ErrorInvalidReportType")

	// Policy
	ErrorPolicyFromContext                    = ErrorKey("ErrorPolicyFromContext")
	ErrorPolicyNotFound                       = ErrorKey("ErrorPolicyNotFound")
	ErrorPolicyLoadingItems                   = ErrorKey("ErrorPolicyLoadingItems")
	ErrorPolicyUpdateInvalidInput             = ErrorKey("ErrorPolicyUpdateInvalidInput")
	ErrorPolicyInviteAlreadyHasHousehold      = ErrorKey("ErrorPolicyInviteAlreadyHasHousehold")
	ErrorPolicyUserInviteCode                 = ErrorKey("ErrorPolicyUserInviteCode")
	ErrorPolicyUserInviteDifferentHouseholdID = ErrorKey("ErrorPolicyUserInviteDifferentHouseholdID")
	ErrorPolicyHasNoHouseholdID               = ErrorKey("ErrorPolicyHasNoHouseholdID")

	// PolicyDependent
	ErrorPolicyDependentCreate = ErrorKey("ErrorPolicyDependentCreate")
	ErrorPolicyDependentDelete = ErrorKey("ErrorPolicyDependentDelete")

	// ClaimItem
	ErrorClaimItemCreateInvalidInput     = ErrorKey("ErrorClaimItemCreateInvalidInput")
	ErrorClaimItemNotRepairable          = ErrorKey("ClaimItemNotRepairable")
	ErrorClaimItemMissingPayoutOption    = ErrorKey("ClaimItemMissingPayoutOption")
	ErrorClaimItemMissingIsRepairable    = ErrorKey("ErrorClaimItemMissingIsRepairable")
	ErrorClaimItemMissingReplaceEstimate = ErrorKey("ClaimItemMissingReplaceEstimate")
	ErrorClaimItemMissingRepairEstimate  = ErrorKey("ClaimItemMissingRepairEstimate")
	ErrorClaimItemMissingFMV             = ErrorKey("ClaimItemMissingFMV")
	ErrorClaimItemInvalidPayoutOption    = ErrorKey("ClaimItemInvalidPayoutOption")
	ErrorClaimInvalidApprover            = ErrorKey("ClaimInvalidApprover")
)
