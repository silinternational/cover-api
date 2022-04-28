package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
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

	polUserIDs := policy.GetPolicyUserIDs(tx, false)

	return renderOk(c, policy.Members.ConvertToPolicyMembers(polUserIDs))
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

// swagger:operation DELETE /policy-members/{id} PolicyMembers PolicyMembersDelete
//
// PolicyMembersDelete
//
// Delete a policy user as long as the related policy has another user. Also,
//   null out the PolicyUserID on related items
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy-member ID
// responses:
//   '204':
//     description: OK but no content in response
func policiesMembersDelete(c buffalo.Context) error {
	policyUser := getReferencedPolicyMemberFromCtx(c)

	if err := policyUser.Delete(c); err != nil {
		return reportError(c, err)
	}

	return c.Render(http.StatusNoContent, nil)

}

// getReferencedPolicyMemberFromCtx pulls the models.PolicyUser resource from context that was put there
// by the AuthZ middleware
func getReferencedPolicyMemberFromCtx(c buffalo.Context) *models.PolicyUser {
	policyUser, ok := c.Value(domain.TypePolicyMember).(*models.PolicyUser)
	if !ok {
		panic("policy user not found in context")
	}
	return policyUser
}
