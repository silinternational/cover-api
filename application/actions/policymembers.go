package actions

import (
	"errors"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /policies/{id}/members PolicyMembers PolicyMembersList
//
// PolicyMembersList
//
// gets the data for all the members of a Policy
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
// responses:
//   '200':
//     description: all policy members
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/PolicyMember"
func policiesListMembers(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)

	policy.LoadMembers(tx, false)

	return renderOk(c, policy.Members.ConvertToPolicyMembers())
}

// swagger:operation POST /policies/{id}/members PolicyMembers PolicyMembersInvite
//
// PolicyMembersInvite
//
// invite new user to co-manage policy
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
//   - name: policy member invite input
//     in: body
//     description: policy user invite input object
//     required: true
//     schema:
//       "$ref": "#/definitions/PolicyUserInviteCreate"
// responses:
//   '204':
//     description: success, no content
//   '400':
//	   description: bad request, check the error and fix your code
func policiesInviteMember(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)

	policy.LoadMembers(tx, false)

	var invite api.PolicyUserInviteCreate
	if err := StrictBind(c, &invite); err != nil {
		return reportError(c, err)
	}

	cUser := models.CurrentUser(c)

	var err error

	if policy.Type == api.PolicyTypeHousehold {
		err = policy.NewHouseholdInvite(tx, invite, cUser)
	} else {
		// make sure user is not already a member of this policy
		if policy.MemberHasEmail(tx, invite.Email) {
			return c.Render(http.StatusNoContent, nil)
		}

		err = policy.NewTeamInvite(tx, invite, cUser)
	}

	if err != nil {
		return reportError(c, err)
	}

	return c.Render(http.StatusNoContent, nil)
}

// swagger:operation DELETE /policies/{id}/members/{user-id} PolicyMembers PolicyMembersDelete
//
// PolicyMembersDelete
//
// Delete a policy user as long as the related policy has another user. Also,
//   switch the accountable person on all related items to a different policy member
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
//   - name: user-id
//     in: path
//     required: true
//     description: user ID of policy member to be deleted
// responses:
//   '204':
//     description: OK but no content in response
func policiesMembersDelete(c buffalo.Context) error {
	tx := models.Tx(c)
	path := c.Request().URL.Path
	pathParts := getPathParts(path)
	partsCount := len(pathParts)

	if partsCount != 4 {
		err := api.NewAppError(errors.New("Bad url path: "+path), api.ErrorValidation, api.CategoryUser)
		return reportError(c, err)
	}

	policy := getReferencedPolicyFromCtx(c)

	userID := uuid.FromStringOrNil(pathParts[3])

	var policyUser models.PolicyUser
	if err := policyUser.FindByPolicyAndUserIDs(tx, policy.ID, userID); err != nil {
		err := api.NewAppError(err, api.ErrorResourceNotFound, api.CategoryUser)
		return reportError(c, err)
	}

	if err := policyUser.Delete(c); err != nil {
		return reportError(c, err)
	}

	return c.Render(http.StatusNoContent, nil)

}
