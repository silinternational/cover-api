package messages

import (
	"testing"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func (ts *TestSuite) Test_ItemSubmittedQueueMessage() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   2,
		ItemsPerPolicy:   2,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	steward0 := models.CreateAdminUsers(db)[models.AppRoleSteward]
	steward1 := models.CreateAdminUsers(db)[models.AppRoleSteward]

	submittedItem := models.UpdateItemStatus(db, f.Items[0], api.ItemCoverageStatusPending, "")

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		data testData
		item models.Item
	}{
		{
			data: testData{
				name:                  "just submitted, not approved",
				wantToEmails:          []interface{}{steward0.EmailOfChoice(), steward1.EmailOfChoice()},
				wantSubjectContains:   "just submitted a new policy item for approval",
				wantInappTextContains: "A new policy item is waiting for your approval",
				wantBodyContains: []string{
					domain.Env.UIURL,
					submittedItem.Name,
					"Coverage is pending your approval for",
				},
			},
			item: submittedItem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.data.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ItemSubmittedQueueMessage(db, tt.item)
			validateNotificationUsers(ts, db, tt.data)

			notfns := models.Notifications{}
			ts.NoError(db.All(&notfns), "error fetching all NotificationUsers for destroy")
			ts.NoError(db.Destroy(&notfns), "error destroying all NotificationUsers")
		})
	}
}

func (ts *TestSuite) Test_ItemAutoApprovedQueueMessage() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   2,
		ItemsPerPolicy:   2,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	steward0 := models.CreateAdminUsers(db)[models.AppRoleSteward]
	steward1 := models.CreateAdminUsers(db)[models.AppRoleSteward]

	approvedItem := models.UpdateItemStatus(db, f.Items[1], api.ItemCoverageStatusApproved, "")

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		data testData
		item models.Item
	}{
		{
			data: testData{
				name:                  "auto approved - stewards",
				wantToEmails:          []interface{}{steward0.EmailOfChoice(), steward1.EmailOfChoice()},
				wantSubjectContains:   "a new policy item that has been auto approved",
				wantInappTextContains: "Coverage on a new policy item was just auto approved",
				wantBodyContains: []string{
					domain.Env.UIURL,
					approvedItem.Name,
					"Coverage has been auto approved for",
				},
			},
			item: approvedItem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.data.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ItemAutoApprovedQueueMessage(db, tt.item)
			validateNotificationUsers(ts, db, tt.data)

			notfns := models.Notifications{}
			ts.NoError(db.All(&notfns), "error fetching all NotificationUsers for destroy")
			ts.NoError(db.Destroy(&notfns), "error destroying all NotificationUsers")
		})
	}
}

func (ts *TestSuite) Test_ItemRevisionQueueMessage() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   2,
		ItemsPerPolicy:   2,
	}

	f := models.CreateItemFixtures(db, fixConfig)
	models.CreateAdminUsers(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	revisionItem := f.Items[0]
	models.UpdateItemStatus(db, revisionItem, api.ItemCoverageStatusRevision, "you can't be serious")

	tests := []testData{
		{
			name:                  "revisions required",
			wantToEmails:          []interface{}{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "Coverage needs attention",
			wantInappTextContains: "Coverage needs attention",
			wantBodyContains: []string{
				domain.Env.UIURL,
				revisionItem.Name,
				revisionItem.StatusReason,
				"we need to clarify a few things on your request",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ItemRevisionQueueMessage(db, revisionItem)
			var notnUsers models.NotificationUsers
			ts.NoError(db.Where("email_address in (?)",
				tt.wantToEmails[0], tt.wantToEmails[1]).All(&notnUsers))

			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ItemDeniedQueueMessage() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   2,
		ItemsPerPolicy:   2,
	}

	f := models.CreateItemFixtures(db, fixConfig)
	models.CreateAdminUsers(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	deniedItem := f.Items[0]
	models.UpdateItemStatus(db, deniedItem, api.ItemCoverageStatusDenied, "this will never fly")

	tests := []testData{
		{
			name:                  "coverage denied",
			wantToEmails:          []interface{}{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "coverage on your new policy item has been denied",
			wantInappTextContains: "coverage on your new policy item has been denied",
			wantBodyContains: []string{
				domain.Env.UIURL,
				deniedItem.Name,
				deniedItem.StatusReason,
				"Coverage was not approved for",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ItemDeniedQueueMessage(db, deniedItem)
			validateNotificationUsers(ts, db, tt)
		})
	}
}
