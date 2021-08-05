package api

import "github.com/gofrs/uuid"

type PolicyDependents []PolicyDependent

type PolicyDependent struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	BirthYear int       `json:"birth_year"`
}

type PolicyDependentInput struct {
	Name      string `json:"name"`
	BirthYear int    `json:"birth_year"`
}
