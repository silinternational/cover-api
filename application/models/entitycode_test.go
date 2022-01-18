package models

import (
	"testing"

	"github.com/silinternational/cover-api/api"
)

func (ms *ModelSuite) TestEntityCode_FindByCode() {
	entityFixture := CreateEntityFixture(ms.DB)

	tests := []struct {
		name string
		code string

		appError *api.AppError
	}{
		{
			name:     "empty string",
			code:     "",
			appError: &api.AppError{Category: api.CategoryUser, Key: api.ErrorNoRows},
		},
		{
			name:     "not found",
			code:     "not a real code",
			appError: &api.AppError{Category: api.CategoryUser, Key: api.ErrorNoRows},
		},
		{
			name:     "good",
			code:     entityFixture.Code,
			appError: nil,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			ec := &EntityCode{
				Code: tt.code,
			}
			err := ec.FindByCode(ms.DB)
			if tt.appError != nil {
				ms.EqualAppError(*tt.appError, err)
				return
			}
			ms.NoError(err)
			ms.Equal(entityFixture.ID, ec.ID, "found wrong entity code")
		})
	}
}
