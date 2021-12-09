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
	"claimIncidentType":             validateClaimIncidentType,
	"claimStatus":                   validateClaimStatus,
	"claimFilePurpose":              validateClaimFilePurpose,
	"payoutOption":                  validatePayoutOption,
	"policyDependentChildBirthYear": validatePolicyDependentChildBirthYear,
	"policyDependentRelationship":   validatePolicyDependentRelationship,
	"policyType":                    validatePolicyType,
	"itemCategoryStatus":            validateItemCategoryStatus,
	"itemCoverageStatus":            validateItemCoverageStatus,
	"ledgerEntryRecordType":         validateLedgerEntryRecordType,
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

func validateClaimIncidentType(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.ClaimIncidentType); ok {
		if value == "" {
			return true
		}
		_, valid := ValidClaimIncidentTypes[value]
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

func validateClaimFilePurpose(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.ClaimFilePurpose); ok {
		_, valid := ValidClaimFilePurpose[value]
		return valid
	}
	return false
}

func validatePayoutOption(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(api.PayoutOption); ok {
		if value == "" {
			return true
		}
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

func validateLedgerEntryRecordType(field validator.FieldLevel) bool {
	if value, ok := field.Field().Interface().(LedgerEntryType); ok {
		_, valid := ValidLedgerEntryTypes[value]
		return valid
	}
	return false
}

func claimStructLevelValidation(sl validator.StructLevel) {
	claim, ok := sl.Current().Interface().(Claim)
	if !ok {
		panic("claimStructLevelValidation registered to a type other than Claim")
	}

	if claim.Status.WasReviewed() {
		if !claim.ReviewerID.Valid {
			sl.ReportError(claim.Status, "reviewer_id", "ReviewerID", "reviewer_required", "")
		}

		if !claim.ReviewDate.Valid {
			sl.ReportError(claim.Status, "review_date", "ReviewDate", "review_date_required", "")
		}
	}
}

func claimItemStructLevelValidation(sl validator.StructLevel) {
	claimItem, ok := sl.Current().Interface().(ClaimItem)
	if !ok {
		panic("claimItemStructLevelValidation registered to a type other than ClaimItem")
	}

	switch claimItem.Claim.Status {
	case api.ClaimStatusRevision, api.ClaimStatusReview1:
		incidentTypePayoutOptions, ok := ValidClaimIncidentTypePayoutOptions[claimItem.Claim.IncidentType]
		if !ok {
			sl.ReportError(claimItem.Claim.IncidentType, "IncidentType", "IncidentType", "invalid Incident type", "")
			return
		}

		if _, ok := incidentTypePayoutOptions[claimItem.PayoutOption]; !ok {
			var options []string
			for k := range incidentTypePayoutOptions {
				options = append(options, string(k))
			}
			sl.ReportError(claimItem.PayoutOption, "payout_option", "PayoutOption",
				fmt.Sprintf("with incident type %s, payout option must be one of: %s",
					claimItem.Claim.IncidentType, strings.Join(options, ", ")), "")
		}

		return
	}
}

func policyStructLevelValidation(sl validator.StructLevel) {
	policy, ok := sl.Current().Interface().(Policy)
	if !ok {
		panic("policyStructLevelValidation registered to a type other than Policy")
	}

	if policy.Type == api.PolicyTypeHousehold {
		if policy.EntityCodeID.String() != HouseholdEntityIDString {
			sl.ReportError(policy.CostCenter, "entity_code_id", "EntityCodeID", "entity_code_id_not_household", "")
		}
		if policy.CostCenter != "" {
			sl.ReportError(policy.CostCenter, "cost_center", "CostCenter", "cost_center_not_permitted", "")
		}
		if policy.Account != "" {
			sl.ReportError(policy.Account, "account", "Account", "account_not_permitted", "")
		}
	} else if policy.Type == api.PolicyTypeTeam {
		if policy.CostCenter == "" {
			sl.ReportError(policy.CostCenter, "cost_center", "CostCenter", "cost_center_required", "")
		}
		if policy.Account == "" {
			sl.ReportError(policy.Account, "account", "Account", "account_not_required", "")
		}
		if policy.HouseholdID.Valid {
			sl.ReportError(policy.HouseholdID, "household_id", "HouseholdID", "household_id_not_permitted", "")
		}
	}
}

func itemStructLevelValidation(sl validator.StructLevel) {
	item, ok := sl.Current().Interface().(Item)
	if !ok {
		panic("itemStructLevelValidation registered to a type other than Item")
	}

	if item.PolicyUserID.Valid && item.PolicyDependentID.Valid {
		sl.ReportError(item.PolicyDependentID, "policy_dependent_id", "PolicyDependentID", "accountable_person_conflict", "")
	}
}

func notificationStructLevelValidation(sl validator.StructLevel) {
	notn, ok := sl.Current().Interface().(Notification)
	if !ok {
		panic("notificationStructLevelValidation registered to a type other than Notification")
	}

	// Body and Subject are both required if InappText is blank or if either
	// of them is present
	if notn.InappText == "" || notn.Body != "" || notn.Subject != "" {
		if notn.Body == "" {
			sl.ReportError(notn.Body, "body", "Body",
				"body_required", "")
		}
		if notn.Subject == "" {
			sl.ReportError(notn.Subject, "subject", "Subject",
				"subject_required", "")
		}
	}
}
