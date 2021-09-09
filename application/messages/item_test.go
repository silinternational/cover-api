package messages

import (
	"testing"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func (ts *TestSuite) Test_ItemSubmittedSend() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    1,
		UsersPerPolicy:      2,
		ClaimsPerPolicy:     1,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	steward0 := models.CreateAdminUsers(db)[models.AppRoleSteward]
	steward1 := models.CreateAdminUsers(db)[models.AppRoleSteward]
	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	submittedItem := models.UpdateItemStatus(db, f.Items[0], api.ItemCoverageStatusPending)
	approvedItem := models.UpdateItemStatus(db, f.Items[1], api.ItemCoverageStatusApproved)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		data testData
		item models.Item
	}{
		{
			data: testData{
				name:         "just submitted, not approved",
				wantToEmails: []string{steward0.EmailOfChoice(), steward1.EmailOfChoice()},
				wantSubjectsContain: []string{
					"just submitted a new policy item for approval",
					"just submitted a new policy item for approval",
				},
			},
			item: submittedItem,
		},
		{
			data: testData{
				name: "auto approved",
				wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice(),
					steward0.EmailOfChoice(), steward1.EmailOfChoice(),
				},
				wantSubjectsContain: []string{
					"your new policy item has been approved",
					"your new policy item has been approved",
					"a new policy item that has been auto approved",
					"a new policy item that has been auto approved",
				},
			},
			item: approvedItem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.data.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ItemSubmittedSend(tt.item, []interface{}{&testEmailer})
			validateEmails(ts, tt.data, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_ItemRevisionSend() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   2,
		ItemsPerPolicy:   2,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	revisionItem := f.Items[0]
	models.UpdateItemStatus(db, revisionItem, api.ItemCoverageStatusRevision)

	testEmailer := notifications.DummyEmailService{}

	tests := []testData{
		{
			name:         "revisions required",
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"changes have been requested on your new policy item",
				"changes have been requested on your new policy item",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ItemRevisionSend(revisionItem, []interface{}{&testEmailer})
			validateEmails(ts, tt, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_ItemDeniedSend() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   2,
		ItemsPerPolicy:   2,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	revisionItem := f.Items[0]
	models.UpdateItemStatus(db, revisionItem, api.ItemCoverageStatusDenied)

	testEmailer := notifications.DummyEmailService{}

	tests := []testData{
		{
			name:         "coverage denied",
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"coverage on your new policy item has been denied",
				"coverage on your new policy item has been denied",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ItemDeniedSend(revisionItem, []interface{}{&testEmailer})
			validateEmails(ts, tt, testEmailer)
		})
	}
}
