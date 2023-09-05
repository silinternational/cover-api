package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation POST /claims/{id}/files ClaimFiles ClaimFilesAttach
// ClaimFilesAttach
//
// attach a File to a Claim
// ---
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: claim ID
//	  - name: claim file input
//	    in: body
//	    description: claim file attach input object
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/ClaimFileAttachInput"
//	responses:
//	  '200':
//	    description: the new ClaimFile
//	    schema:
//	      "$ref": "#/definitions/ClaimFile"
func claimFilesAttach(c buffalo.Context) error {
	var input api.ClaimFileAttachInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)

	claim := getReferencedClaimFromCtx(c)
	claimFile, err := claim.AttachFile(tx, input)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, claimFile.ConvertToAPI(tx))
}

// swagger:operation DELETE /claim-files/{id} ClaimFiles ClaimFilesDelete
// ClaimFilesDelete
//
// Delete a ClaimFile and its associated File in the db and on S3
// ---
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: claim file ID
//	responses:
//	  '204':
//	    description: OK but no content in response
func claimFilesDelete(c buffalo.Context) error {
	tx := models.Tx(c)
	cFile := getReferencedClaimFileFromCtx(c)
	cFile.Destroy(tx)
	return c.Render(http.StatusNoContent, nil)
}

// getReferencedClaimFileFromCtx pulls the models.ClaimFile resource from context that was put there
// by the AuthZ middleware
func getReferencedClaimFileFromCtx(c buffalo.Context) *models.ClaimFile {
	file, ok := c.Value(domain.TypeClaimFile).(*models.ClaimFile)
	if !ok {
		panic("claim file not found in context")
	}
	return file
}
