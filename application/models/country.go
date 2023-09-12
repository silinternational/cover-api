package models

import (
	"strings"

	"github.com/gobuffalo/pop/v6"

	"github.com/silinternational/cover-api/api"
)

// swagger:model
type Country struct {
	ID   string `db:"code" json:"code"`
	Name string `db:"name" json:"name"`
}

// swagger:model
type Countries []Country

func (c *Countries) All(tx *pop.Connection) error {
	err := tx.All(c)
	if err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	return nil
}

func (c *Country) FindByCode(tx *pop.Connection, code string) error {
	err := tx.Where("code = ?", strings.ToUpper(code)).First(c)
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
