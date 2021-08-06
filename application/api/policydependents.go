package api

import "github.com/gofrs/uuid"

type PolicyDependentRelationship string

const (
	PolicyDependentRelationshipSpouse = PolicyDependentRelationship("Spouse")
	PolicyDependentRelationshipChild  = PolicyDependentRelationship("Child")
)

type PolicyDependents []PolicyDependent

type PolicyDependent struct {
	ID             uuid.UUID                   `json:"id"`
	Name           string                      `json:"name"`
	Relationship   PolicyDependentRelationship `json:"relationship"`
	Location       string                      `json:"location"`
	ChildBirthYear int                         `json:"child_birth_year"`
}

type PolicyDependentInput struct {
	Name           string                      `json:"name"`
	Relationship   PolicyDependentRelationship `json:"relationship"`
	Location       string                      `json:"location"`
	ChildBirthYear int                         `json:"child_birth_year"`
}
