package api

import (
	"github.com/gofrs/uuid"
)

// swagger:model
type AccountablePerson struct {
	// ID that can reference either a User or a PolicyDependent
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// full name
	Name string `json:"name"`

	// country where person is located
	Country string `json:"country"`

	// country code (ISO-3166 alpha-3) where person is located
	CountryCode string `json:"country_code"`
}
