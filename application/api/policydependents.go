package api

import "github.com/gofrs/uuid"

// PolicyDependentRelationship
//
// may be one of: Spouse, Child
//
// swagger:model
type PolicyDependentRelationship string

const (
	PolicyDependentRelationshipNone   = PolicyDependentRelationship("None")
	PolicyDependentRelationshipSpouse = PolicyDependentRelationship("Spouse")
	PolicyDependentRelationshipChild  = PolicyDependentRelationship("Child")
)

// swagger:model
type PolicyDependents []PolicyDependent

// swagger:model
type PolicyDependent struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// dependent name
	Name string `json:"name"`

	// dependent relationship
	Relationship PolicyDependentRelationship `json:"relationship"`

	// dependent location
	Country string `json:"country"`

	// birth year of child
	ChildBirthYear int `json:"child_birth_year"`
}

// swagger:model
type PolicyDependentInput struct {
	// dependent name
	Name string `json:"name"`

	// dependent relationship, one of: Spouse, Child
	Relationship PolicyDependentRelationship `json:"relationship"`

	// dependent location
	Country string `json:"country"` // TODO: look up country code

	// birth year of child
	ChildBirthYear int `json:"child_birth_year"`
}
