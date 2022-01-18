package models

import (
	"fmt"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
)

const HouseholdEntityIDString = "5f181e39-0a2a-49ac-8796-2f3a3de9fcbd"

var householdEntityID uuid.UUID

type EntityCodes []EntityCode

type EntityCode struct {
	ID            uuid.UUID `db:"id"`
	Code          string    `db:"code"`
	Name          string    `db:"name"`
	Active        bool      `db:"active"`
	IncomeAccount string    `db:"income_account"`
	ParentEntity  string    `db:"parent_entity"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (ec *EntityCode) Create(tx *pop.Connection) error {
	ec.ID = GetV5UUID(ec.Code)
	return create(tx, ec)
}

func (ec *EntityCode) FindByCode(tx *pop.Connection) error {
	err := tx.Where("code = ?", ec.Code).First(ec)
	if err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	return nil
}

func EntityCodeID(code string) uuid.UUID {
	return GetV5UUID(code)
}

func (ec *EntityCodes) ConvertToAPI(tx *pop.Connection) []api.EntityCode {
	entityCodes := make([]api.EntityCode, len(*ec))
	for i, cc := range *ec {
		entityCodes[i] = cc.ConvertToAPI(tx)
	}
	return entityCodes
}

func (ec *EntityCodes) AllActive(tx *pop.Connection) error {
	err := tx.Where("active = true").Where("id != ?", HouseholdEntityIDString).Order("code").All(ec)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

func (ec *EntityCode) ConvertToAPI(tx *pop.Connection) api.EntityCode {
	return api.EntityCode{
		ID:   ec.ID,
		Code: ec.Code,
		Name: fmt.Sprintf("%s - %s", ec.Code, ec.Name),
	}
}

func HouseholdEntityID() uuid.UUID {
	return householdEntityID
}
