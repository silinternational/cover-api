package models

import (
	"net/http"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
)

type EntityCodes []EntityCode

type EntityCode struct {
	ID   uuid.UUID `db:"id"`
	Code string    `db:"code"`
	Name string    `db:"name"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (ec *EntityCode) Create(tx *pop.Connection) error {
	return create(tx, ec)
}

func (ec *EntityCode) GetID() uuid.UUID {
	return ec.ID
}

func (ec *EntityCode) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(ec, id)
}

func (ec *EntityCode) FindByCode(tx *pop.Connection, code string) error {
	err := tx.Where("code = ?", code).First(ec)
	return appErrorFromDB(err, api.ErrorNoRows)
}

func (ec *EntityCode) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	if perm == PermissionList {
		return true
	}

	return false
}

func (ec *EntityCodes) ConvertToAPI(tx *pop.Connection) interface{} {
	entityCodes := make([]api.EntityCode, len(*ec))
	for i, cc := range *ec {
		entityCodes[i] = cc.ConvertToAPI(tx)
	}
	return entityCodes
}

func (ec *EntityCode) ConvertToAPI(tx *pop.Connection) api.EntityCode {
	return api.EntityCode{
		ID:   ec.ID,
		Code: ec.Code,
		Name: ec.Name,
	}
}
