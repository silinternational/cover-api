package models

import (
	"database/sql"
	"testing"

	"github.com/gofrs/uuid"
)

func (ms *ModelSuite) TestPolicyUser_Delete() {

	f1 := CreateItemFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 4, ItemsPerPolicy: 2, UsersPerPolicy: 2})
	user := f1.Users[0]

	policyWithItems := f1.Policies[0]
	polUserIDs := policyWithItems.GetPolicyUserIDs(ms.DB, false)

	var polUserExtraForPolicy PolicyUser
	ms.NoError(polUserExtraForPolicy.FindByID(ms.DB, polUserIDs[0]), "error fetching PolicyUser fixture")

	var itemsForUser Items
	ms.NoError(ms.DB.Where("policy_user_id = ?", polUserExtraForPolicy.UserID).All(&itemsForUser),
		"error fetching the item fixtures related to the Policy User")

	f2 := CreateItemFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 4, ItemsPerPolicy: 1, UsersPerPolicy: 1})
	policyOnly1User := f2.Policies[0]

	polUserIDs = policyOnly1User.GetPolicyUserIDs(ms.DB, false)
	var polUserOnly1ForPolicy PolicyUser
	ms.NoError(polUserOnly1ForPolicy.FindByID(ms.DB, polUserIDs[0]), "error fetching PolicyUser fixture")

	ctx := CreateTestContext(user)

	tests := []struct {
		name            string
		polUser         PolicyUser
		items           Items
		wantErrContains string
	}{
		{
			name:            "only one policy user for the policy",
			polUser:         polUserOnly1ForPolicy,
			wantErrContains: "may not delete the last of a policy's users",
		},
		{
			name:            "policy has more than one policy user",
			polUser:         polUserExtraForPolicy,
			items:           itemsForUser,
			wantErrContains: "",
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			err := tt.polUser.Delete(ctx)
			if tt.wantErrContains != "" {
				ms.Error(err)
				ms.Contains(err.Error(), tt.wantErrContains, "incorrect error")
				return
			}

			ms.NoError(err)

			// Ensure it got deleted
			var polUser PolicyUser
			dbErr := polUser.FindByID(ms.DB, tt.polUser.ID)
			ms.Error(dbErr, "expected to have a SQL error for no rows")
			ms.Contains(dbErr.Error(), sql.ErrNoRows.Error(),
				"incorrect error trying to fetch deleted PolicyUser from db")

			// Check that items have no PolicyUserID
			for _, item := range itemsForUser {
				ms.False(item.PolicyUserID.Valid,
					"PolicyUserID was not nulled out on item %s", item.ID.String())
				ms.Equal(uuid.Nil, item.PolicyUserID.UUID,
					"PolicyUserID was not nulled out on item %s", item.ID.String())
			}
		})
	}
}
