package models

import (
	"fmt"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
)

const HouseholdEntityIDString = "5f181e39-0a2a-49ac-8796-2f3a3de9fcbd"

var HouseholdEntityID uuid.UUID

type EntityCodes []EntityCode

type EntityCode struct {
	ID            uuid.UUID `db:"id"`
	Code          string    `db:"code"`
	Name          string    `db:"name"`
	Active        bool      `db:"active"`
	IncomeAccount string    `db:"income_account"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (ec *EntityCode) Create(tx *pop.Connection) error {
	return create(tx, ec)
}

func (ec *EntityCode) FindByCode(tx *pop.Connection, code string) error {
	err := tx.Where("code = ?", code).First(ec)
	return appErrorFromDB(err, api.ErrorNoRows)
}

func (ec *EntityCodes) ConvertToAPI(tx *pop.Connection) []api.EntityCode {
	entityCodes := make([]api.EntityCode, len(*ec))
	for i, cc := range *ec {
		entityCodes[i] = cc.ConvertToAPI(tx)
	}
	return entityCodes
}

func (ec *EntityCodes) AllActive(tx *pop.Connection) error {
	err := tx.Where("active = true").Order("code").All(ec)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

func (ec *EntityCode) ConvertToAPI(tx *pop.Connection) api.EntityCode {
	return api.EntityCode{
		ID:   ec.ID,
		Code: ec.Code,
		Name: fmt.Sprintf("%s - %s", ec.Code, ec.Name),
	}
}
