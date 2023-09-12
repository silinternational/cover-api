package models

import (
	"github.com/gobuffalo/pop/v6"

	"github.com/silinternational/cover-api/api"
)

// swagger:model
type Country struct {
	Code string `db:"code" json:"code"`
	Name string `db:"name" json:"name"`
}

// swagger:model
type Countries []Country

func (c *Countries) All(tx *pop.Connection) error {
	err := tx.All(c)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

func (c *Country) FindByCode(tx *pop.Connection, code string) error {
	err := tx.Where("code = ?", code).First(c)
	if err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	return nil
}

func (c *Country) FindByName(tx *pop.Connection, name string) error {
	err := tx.Where("name = ?", name).First(c)
	if err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	return nil
}
