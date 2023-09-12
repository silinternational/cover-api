package models

import (
	"strings"
	"testing"

	"github.com/silinternational/cover-api/api"
)

func (ms *ModelSuite) TestCountry_FindByCode() {
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
			code:     "not a good code",
			appError: &api.AppError{Category: api.CategoryUser, Key: api.ErrorNoRows},
		},
		{
			name:     "mixed case",
			code:     FakeCountries[0][0:3],
			appError: nil,
		},
		{
			name:     "upper case",
			code:     strings.ToUpper(FakeCountries[1][0:3]),
			appError: nil,
		},
		{
			name:     "lower case",
			code:     strings.ToLower(FakeCountries[2][0:3]),
			appError: nil,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			country := &Country{}
			err := country.FindByCode(ms.DB, tt.code)
			if tt.appError != nil {
				ms.EqualAppError(*tt.appError, err)
				return
			}
			ms.NoError(err)
			ms.Equal(strings.ToUpper(tt.code), country.ID, "found wrong country")
		})
	}
}

func (ms *ModelSuite) TestCountry_FindByName() {
	tests := []struct {
		name    string
		country string

		appError *api.AppError
	}{
		{
			name:     "empty string",
			country:  "",
			appError: &api.AppError{Category: api.CategoryUser, Key: api.ErrorNoRows},
		},
		{
			name:     "not found",
			country:  "not a good country",
			appError: &api.AppError{Category: api.CategoryUser, Key: api.ErrorNoRows},
		},
		{
			name:     "found",
			country:  FakeCountries[0],
			appError: nil,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			country := &Country{}
			err := country.FindByName(ms.DB, tt.country)
			if tt.appError != nil {
				ms.EqualAppError(*tt.appError, err)
				return
			}
			ms.NoError(err)
			ms.Equal(tt.country, country.Name, "found wrong country")
		})
	}
}
