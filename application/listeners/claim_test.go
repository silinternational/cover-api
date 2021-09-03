package listeners

import (
	"testing"

	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func (ts *TestSuite) Test_claimSubmitted() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    1,
		UsersPerPolicy:      2,
		ClaimsPerPolicy:     2,
		ClaimItemsPerClaim:  1,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	steward := models.CreateAdminUser(db)

	review1Claim := f.Claims[0]
	models.UpdateClaimStatus(db, review1Claim, api.ClaimStatusReview1)

	review2Claim := f.Claims[1]
	models.UpdateClaimStatus(db, review2Claim, api.ClaimStatusReview2)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name                string
		event               events.Event
		wantToEmails        []string
		wantSubjectContains string
	}{
		{
			name: "submitted to review1",
			event: events.Event{
				Kind:    domain.EventApiClaimSubmitted,
				Payload: getTestPayload(review1Claim.ID, &testEmailer),
			},
			wantToEmails:        []string{steward.EmailOfChoice()},
			wantSubjectContains: "just submitted a new claim for approval",
		},
		{
			name: "submitted to review2",
			event: events.Event{
				Kind:    domain.EventApiClaimSubmitted,
				Payload: getTestPayload(review2Claim.ID, &testEmailer),
			},
			wantToEmails:        []string{steward.EmailOfChoice()},
			wantSubjectContains: "just resubmitted a claim for approval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()

			claimSubmitted(tt.event)

			msgs := testEmailer.GetSentMessages()
			ts.Len(msgs, 1, "incorrect message count")

			gotTos := testEmailer.GetAllToAddresses()
			ts.Equal(tt.wantToEmails, gotTos)

			ts.Contains(msgs[0].Subject, tt.wantSubjectContains, "incorrect email subject")

		})
	}
}
