package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /policies Policies PoliciesList
//
// PoliciesList
//
// Get the data for all the user's Policies if the user is not an admin. If called by an admin, returns all Policies
// in the system, limited by query parameters.
//
// ---
// parameters:
// - name: limit
//   in: query
//   required: false
//   description: number of records to return, minimum 1, maximum 50, default 10
// - name: search
//   in: query
//   required: false
//   description: search text to find across fields (name, household_id, cost_center, and all members' first and last names)
// - name: filter
//   in: query
//   required: false
//   description: comma-separated list of search pairs like "field:text". Presently, only meta-field 'active' is supported
// responses:
//   '200':
//     description: all policies
//     schema:
//       type: object
//       properties:
//         meta:
//           "$ref": "#/definitions/Meta"
//         data:
//           "$ref": "#/definitions/Policies"
func policiesList(c buffalo.Context) error {
	user := models.CurrentUser(c)

	if user.IsAdmin() {
		return policiesListAdmin(c)
	}

	return policiesListCustomer(c)
}

func policiesListAdmin(c buffalo.Context) error {
	tx := models.Tx(c)
	var policies models.Policies

	p := api.NewQueryParams(c.Params())
	q, err := policies.Query(tx, p)
	if err != nil {
		return reportError(c, err)
	}

	response := api.ListResponse{
		Data: policies.ConvertToAPI(tx),
		Meta: api.Meta{Paginator: q.Paginator},
	}

	return renderOk(c, response)
}

func policiesListCustomer(c buffalo.Context) error {
	tx := models.Tx(c)
	user := models.CurrentUser(c)

	user.LoadPolicies(tx, false)

	response := api.ListResponse{
		Data: user.Policies.ConvertToAPI(tx),
	}

	return renderOk(c, response)
}

// swagger:operation GET /policies/{id} Policies PoliciesView
//
// PoliciesView
//
// gets the data for a specific policy
//
// ---
// responses:
//   '200':
//     description: a policy
//     schema:
//       "$ref": "#/definitions/Policy"
func policiesView(c buffalo.Context) error {
	policy := getReferencedPolicyFromCtx(c)

	return renderOk(c, policy.ConvertToAPI(models.Tx(c), true))
}

// swagger:operation POST /policies Policies PoliciesCreateTeam
//
// PoliciesCreateTeam
//
// create a new Policy with type Team
//
// ---
// parameters:
//   - name: policy input
//     in: body
//     description: policy create input object
//     required: true
//     schema:
//       "$ref": "#/definitions/PolicyCreate"
// responses:
//   '200':
//     description: the new Policy
//     schema:
//       "$ref": "#/definitions/Policy"
func policiesCreateTeam(c buffalo.Context) error {
	var input api.PolicyCreate
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)
	user := models.CurrentUser(c)

	policy := models.Policy{
		Name:          input.Name,
		CostCenter:    input.CostCenter,
		Account:       input.Account,
		AccountDetail: input.AccountDetail,
		EntityCodeID:  models.EntityCodeID(input.EntityCode),
	}

	if err := policy.CreateTeam(tx, user); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, policy.ConvertToAPI(tx, true))
}

// swagger:operation PUT /policies/{id} Policies PoliciesUpdate
//
// PoliciesUpdate
//
// update a policy
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: policy ID
//   - name: policy update input
//     in: body
//     description: policy update input object
//     required: true
//     schema:
//       "$ref": "#/definitions/PolicyUpdate"
// responses:
//   '200':
//     description: updated Policy
//     schema:
//       "$ref": "#/definitions/Policy"
func policiesUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)

	var update api.PolicyUpdate
	if err := StrictBind(c, &update); err != nil {
		return reportError(c, err)
	}

	switch policy.Type {
	case api.PolicyTypeHousehold:
		if update.HouseholdID != nil {
			policy.HouseholdID = nulls.NewString(*update.HouseholdID)
		}
		policy.CostCenter = ""
		policy.Account = ""
		policy.AccountDetail = ""
		policy.EntityCodeID = models.HouseholdEntityID()
	case api.PolicyTypeTeam:
		policy.HouseholdID = nulls.String{}
		policy.CostCenter = update.CostCenter
		policy.Account = update.Account
		policy.AccountDetail = update.AccountDetail
		policy.EntityCodeID = models.EntityCodeID(update.EntityCode)
	}

	policy.Name = update.Name

	if err := policy.Update(c); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, policy.ConvertToAPI(tx, true))
}

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

	// make sure user is not already a member of this policy
	if policy.MemberHasEmail(tx, invite.Email) {
		return c.Render(http.StatusNoContent, nil)
	}

	// check if user already exists
	var user models.User
	if err := user.FindByEmail(tx, invite.Email); domain.IsOtherThanNoRows(err) {
		return reportError(c, err)
	}
	if user.ID != uuid.Nil {
		pUser := models.PolicyUser{
			PolicyID: policy.ID,
			UserID:   user.ID,
		}
		if err := pUser.Create(tx); err != nil {
			return reportError(c, err)
		}

		return c.Render(http.StatusNoContent, nil)
	}

	// create invite
	cUser := models.CurrentUser(c)
	puInvite := models.PolicyUserInvite{
		PolicyID:       policy.ID,
		Email:          invite.Email,
		InviteeName:    invite.Name,
		InviterName:    cUser.Name(),
		InviterEmail:   cUser.Email,
		InviterMessage: invite.InviterMessage,
	}
	if err := puInvite.Create(tx); err != nil {
		return reportError(c, err)
	}

	return c.Render(http.StatusNoContent, nil)
}

// getReferencedPolicyFromCtx pulls the models.Policy resource from context that was put there
// by the AuthZ middleware
func getReferencedPolicyFromCtx(c buffalo.Context) *models.Policy {
	policy, ok := c.Value(domain.TypePolicy).(*models.Policy)
	if !ok {
		panic("policy not found in context")
	}
	return policy
}
