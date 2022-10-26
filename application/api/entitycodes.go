package api

import (
	"github.com/gofrs/uuid"
)

// swagger:model
type EntityCodes []EntityCode

// swagger:model
type EntityCode struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// Code is a unique three-letter identifier for an accounting entity
	Code string `json:"code"`

	// Name is a succinct description of the entity code.
	Name string `json:"name"`

	// Active set to true allows it to be displayed in the selection list for policies. Only visible to admins.
	Active *bool `json:"active,omitempty"`

	// IncomeAccount is the account use for income transactions. Only visible to admins.
	IncomeAccount *string `json:"income_account,omitempty"`

	// ParentEntity is the parent entity code for grouping in reports. Only visible to admins.
	ParentEntity *string `json:"parent_entity,omitempty"`
}

// swagger:model
type EntityCodeInput struct {
	// Name is a succinct description of the entity code.
	Name string `json:"name"`

	// Active set to true allows it to be displayed in the selection list for policies. Only visible to admins.
	Active bool `json:"active"`

	// IncomeAccount is the account use for income transactions. Only visible to admins.
	IncomeAccount string `json:"income_account"`

	// ParentEntity is the parent entity code for grouping in reports. Only visible to admins.
	ParentEntity string `json:"parent_entity"`
}
