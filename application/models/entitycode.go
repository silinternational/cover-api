package models

import (
	"net/http"
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

func (ec *EntityCode) Update(tx *pop.Connection) error {
	return update(tx, ec)
}

// GetID returns the EntityCode ID. For Authable interface.
func (ec *EntityCode) GetID() uuid.UUID {
	return ec.ID
}

// FindByID returns the EntityCode identified by the id given. For Authable interface.
func (ec *EntityCode) FindByID(tx *pop.Connection, id uuid.UUID) error {
	err := tx.Find(ec, id)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

// IsActorAllowedTo returns true if the given actor is allowed the given permission. For Authable interface.
func (ec *EntityCode) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, res SubResource, request *http.Request) bool {
	return actor.IsAdmin() || perm == PermissionList
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

func (ec *EntityCodes) ConvertToAPI(tx *pop.Connection, admin bool) []api.EntityCode {
	entityCodes := make([]api.EntityCode, len(*ec))
	for i, cc := range *ec {
		entityCodes[i] = cc.ConvertToAPI(tx, admin)
	}
	return entityCodes
}

func (ec *EntityCodes) All(tx *pop.Connection) error {
	err := tx.All(ec)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

func (ec *EntityCodes) AllActive(tx *pop.Connection) error {
	err := tx.Where("active = true").Where("id != ?", HouseholdEntityIDString).Order("code").All(ec)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

// ConvertToAPI adapts an EntityCode model record to API model. If admin is true, all fields are hydrated.
func (ec *EntityCode) ConvertToAPI(tx *pop.Connection, admin bool) api.EntityCode {
	code := api.EntityCode{
		ID:   ec.ID,
		Code: ec.Code,
		Name: ec.Name,
	}
	if admin {
		code.IncomeAccount = &ec.IncomeAccount
		code.Active = &ec.Active
		code.ParentEntity = &ec.ParentEntity
	}
	return code
}

func HouseholdEntityID() uuid.UUID {
	return householdEntityID
}

func (ec *EntityCode) UpdateFromAPI(tx *pop.Connection, input api.EntityCodeInput) error {
	ec.Name = input.Name
	ec.Active = input.Active
	ec.IncomeAccount = input.IncomeAccount
	ec.ParentEntity = input.ParentEntity
	return ec.Update(tx)
}
