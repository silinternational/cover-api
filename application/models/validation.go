package models

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gobuffalo/validate/v3"
)

// Model validation tool
var mValidate *validator.Validate

var validationTypes = map[string]func(validator.FieldLevel) bool{
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
	if pt, ok := field.Field().Interface().(PolicyType); ok {
		_, valid := ValidPolicyTypes[pt]
		return valid
	}
	return false
}

func validateItemCategoryStatus(field validator.FieldLevel) bool {
	if pt, ok := field.Field().Interface().(ItemCategoryStatus); ok {
		_, valid := ValidItemCategoryStatuses[pt]
		return valid
	}
	return false
}

func validateItemCoverageStatus(field validator.FieldLevel) bool {
	if pt, ok := field.Field().Interface().(ItemCoverageStatus); ok {
		_, valid := ValidItemCoverageStatuses[pt]
		return valid
	}
	return false
}
