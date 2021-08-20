package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gobuffalo/validate/v3"

	"github.com/silinternational/riskman-api/api"
)

// Model validation tool
var mValidate *validator.Validate

var fieldValidators = map[string]func(validator.FieldLevel) bool{
	"appRole":                       validateAppRole,
	"claimEventType":                validateClaimEventType,
	"claimStatus":                   validateClaimStatus,
	"claimItemStatus":               validateClaimItemStatus,
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
	claim := sl.Current().Interface().(Claim)

	if claim.Status != api.ClaimStatusApproved && claim.Status != api.ClaimStatusDenied {
		return
	}

	if !claim.ReviewerID.Valid {
		sl.ReportError(claim.Status, "reviewer_id", "ReviewerID", "reviewer_required", "foo")
	}

	if !claim.ReviewDate.Valid {
		sl.ReportError(claim.Status, "review_date", "ReviewDate", "review_date_required", "")
	}
}

func claimItemStructLevelValidation(sl validator.StructLevel) {
	claimItem := sl.Current().Interface().(ClaimItem)

	if claimItem.Status != api.ClaimItemStatusApproved && claimItem.Status != api.ClaimItemStatusDenied {
		return
	}

	if !claimItem.ReviewerID.Valid {
		sl.ReportError(claimItem.Status, "reviewer_id", "ReviewerID", "reviewer_required", "foo")
	}

	if !claimItem.ReviewDate.Valid {
		sl.ReportError(claimItem.Status, "review_date", "ReviewDate", "review_date_required", "")
	}
}
