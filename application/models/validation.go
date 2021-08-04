package models

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gobuffalo/validate/v3"
	"github.com/silinternational/riskman-api/api"
)

// Model validation tool
var mValidate *validator.Validate

var validationTypes = map[string]func(validator.FieldLevel) bool{
	"appRole":            validateAppRole,
	"policyType":         validatePolicyType,
	"itemCategoryStatus": validateItemCategoryStatus,
	"itemCoverageStatus": validateItemCoverageStatus,
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
