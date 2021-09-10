package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gobuffalo/validate/v3"

	"github.com/silinternational/cover-api/api"
)

// Model validation tool
var mValidate *validator.Validate

var fieldValidators = map[string]func(validator.FieldLevel) bool{
	"appRole":                       validateAppRole,
	"claimEventType":                validateClaimEventType,
	"claimStatus":                   validateClaimStatus,
	"claimItemStatus":               validateClaimItemStatus,
	"claimFilePurpose":              validateClaimFilePurpose,
	"payoutOption":                  validatePayoutOption,
	"policyDependentChildBirthYear": validatePolicyDependentChildBirthYear,
	"policyDependentRelationship":   validatePolicyDependentRelationship,
	"policyType":                    validatePolicyType,
	"itemCategoryStatus":            validateItemCategoryStatus,
	"itemCoverageStatus":            validateItemCoverageStatus,
}

func validateModel(m interface{}) *validate.Errors {
	vErrs := validate.NewErrors()

	if err := mValidate.Struct(m); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			vErrs.Add(err.StructNamespace(), err.Error())
		}
	}
	return vErrs
}

// flattenPopErrors - pop validation errors are complex structures, this flattens them to a simple string
func flattenPopErrors(popErrs *validate.Errors) string {
	var msgs []string
	for key, val := range popErrs.Errors {
		msgs = append(msgs, fmt.Sprintf("%s: %s", key, strings.Join(val, ", ")))
	}
	msg := strings.Join(msgs, " |")
	return msg
}

func validateClaimEventType(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.ClaimEventType); ok {
		_, valid := ValidClaimEventTypes[value]
		return valid
	}
	return false
}

func validateClaimStatus(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.ClaimStatus); ok {
		_, valid := ValidClaimStatus[value]
		return valid
	}
	return false
}

func validateClaimItemStatus(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.ClaimItemStatus); ok {
		_, valid := ValidClaimItemStatus[value]
		return valid
	}
	return false
}

func validateClaimFilePurpose(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.ClaimFilePurpose); ok {
		_, valid := ValidClaimFilePurpose[value]
		return valid
	}
	return false
}

func validatePayoutOption(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.PayoutOption); ok {
		_, valid := ValidPayoutOptions[value]
		return valid
	}
	return false
}

func validatePolicyDependentChildBirthYear(field validator.FieldLevel) bool {
	year := int(field.Field().Int())
	return year <= time.Now().UTC().Year()
}

func validatePolicyDependentRelationship(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.PolicyDependentRelationship); ok {
		_, valid := ValidPolicyDependentRelationships[value]
		return valid
	}
	return false
}

func validatePolicyType(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.PolicyType); ok {
		_, valid := ValidPolicyTypes[value]
		return valid
	}
	return false
}

func validateAppRole(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(UserAppRole); ok {
		_, valid := validUserAppRoles[value]
		return valid
	}
	return false
}

func validateItemCategoryStatus(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.ItemCategoryStatus); ok {
		_, valid := ValidItemCategoryStatuses[value]
		return valid
	}
	return false
}

func validateItemCoverageStatus(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.ItemCoverageStatus); ok {
		_, valid := ValidItemCoverageStatuses[value]
		return valid
	}
	return false
}

func claimStructLevelValidation(sl validator.StructLevel) {
	claim, ok := sl.Current().Interface().(Claim)
	if !ok {
		panic("claimStructLevelValidation registered to a type other than Claim")
	}

	if claim.Status != api.ClaimStatusApproved && claim.Status != api.ClaimStatusDenied {
		return
	}

	if !claim.ReviewerID.Valid {
		sl.ReportError(claim.Status, "reviewer_id", "ReviewerID", "reviewer_required", "")
	}

	if !claim.ReviewDate.Valid {
		sl.ReportError(claim.Status, "review_date", "ReviewDate", "review_date_required", "")
	}
}

func claimItemStructLevelValidation(sl validator.StructLevel) {
	claimItem, ok := sl.Current().Interface().(ClaimItem)
	if !ok {
		panic("claimItemStructLevelValidation registered to a type other than ClaimItem")
	}

	if claimItem.Status == api.ClaimItemStatusPending || claimItem.Status == api.ClaimItemStatusDraft {
		switch claimItem.Claim.EventType {
		case api.ClaimEventTypeEvacuation:
			if claimItem.PayoutOption != api.PayoutOptionFixedFraction {
				sl.ReportError(claimItem.PayoutOption, "payout_option", "PayoutOption",
					"payout_option_must_be_fixed_fraction", "")
			}
		case api.ClaimEventTypeTheft:
			if claimItem.PayoutOption != api.PayoutOptionFMV && claimItem.PayoutOption != api.PayoutOptionReplacement {
				sl.ReportError(claimItem.PayoutOption, "payout_option", "PayoutOption",
					"payout_option_must_be_fmv_or_replacement", "")
			}
		default:
			if claimItem.PayoutOption == api.PayoutOptionFixedFraction {
				sl.ReportError(claimItem.PayoutOption, "payout_option", "PayoutOption",
					"payout_option_must_not_be_fixed_fraction", "")
			}
		}
		return

	}

	if !claimItem.ReviewerID.Valid {
		sl.ReportError(claimItem.Status, "reviewer_id", "ReviewerID", "reviewer_required", "")
	}

	if !claimItem.ReviewDate.Valid {
		sl.ReportError(claimItem.Status, "review_date", "ReviewDate", "review_date_required", "")
	}
}

func policyStructLevelValidation(sl validator.StructLevel) {
	policy, ok := sl.Current().Interface().(Policy)
	if !ok {
		panic("policyStructLevelValidation registered to a type other than Policy")
	}

	if policy.Type == api.PolicyTypeHousehold && !policy.HouseholdID.Valid {
		sl.ReportError(policy.HouseholdID, "household_id", "HouseholdID", "household_id_required", "")
	}
}
