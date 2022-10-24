package actions

import (
	"github.com/gobuffalo/buffalo"
	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /entity-codes EntityCodes EntityCodesList
//
// EntityCodesList
//
// list Entity Codes
//
// ---
// responses:
//   '200':
//     description: list of Entity Codes
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/EntityCodes"
func entityCodesList(c buffalo.Context) error {
	tx := models.Tx(c)
	actor := models.CurrentUser(c)
	var entityCodes models.EntityCodes
	var err error
	if actor.IsAdmin() {
		err = entityCodes.All(tx)
	} else {
		err = entityCodes.AllActive(tx)
	}
	if err != nil {
		return reportError(c, err)
	}
	return renderOk(c, entityCodes.ConvertToAPI(tx, actor.IsAdmin()))
}

// swagger:operation GET /entity-codes/{id} EntityCodes EntityCodesView
//
// EntityCodesView
//
// get a single Entity Code
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: entity code ID
// responses:
//  '200':
//    description: an Entity Code record
//    schema:
//      "$ref": "#/definitions/EntityCode"
func entityCodesView(c buffalo.Context) error {
	e := getReferencedEntityCodeFromCtx(c)
	return renderEntityCode(c, *e)
}

// swagger:operation PUT /entity-codes/{id} EntityCodes EntityCodesUpdate
//
// EntityCodesUpdate
//
// update a Entity Code
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: entity code ID
//   - name: entity code update input
//     in: body
//     description: entity code update input object
//     required: true
//     schema:
//       "$ref": "#/definitions/EntityCodeInput"
// responses:
//  '200':
//    description: an Entity Code record
//    schema:
//      "$ref": "#/definitions/EntityCode"
func entityCodesUpdate(c buffalo.Context) error {
	e := getReferencedEntityCodeFromCtx(c)
	var input api.EntityCodeInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}
	if err := e.UpdateFromAPI(models.Tx(c), input); err != nil {
		return err
	}
	return renderEntityCode(c, *e)
}

// getReferencedEntityCodeFromCtx pulls the models.EntityCode resource from context that was put there
// by the AuthZ middleware
func getReferencedEntityCodeFromCtx(c buffalo.Context) *models.EntityCode {
	entityCode, ok := c.Value(domain.TypeEntityCode).(*models.EntityCode)
	if !ok {
		panic("entityCode not found in context")
	}
	return entityCode
}

func renderEntityCode(c buffalo.Context, e models.EntityCode) error {
	tx := models.Tx(c)
	user := models.CurrentUser(c)
	return renderOk(c, e.ConvertToAPI(tx, user.IsAdmin()))
}
