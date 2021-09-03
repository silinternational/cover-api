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

func (ts *TestSuite) Test_claimRevision() {
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

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	revisionClaim := f.Claims[0]
	models.UpdateClaimStatus(db, revisionClaim, api.ClaimStatusRevision)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name                string
		event               events.Event
		wantToEmails        []string
		wantSubjectsContain []string
	}{
		{
			name: "revisions required",
			event: events.Event{
				Kind:    domain.EventApiClaimRevision,
				Payload: getTestPayload(revisionClaim.ID, &testEmailer),
			},
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"changes have been requested on your claim",
				"changes have been requested on your claim",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()

			claimRevision(tt.event)

			wantCount := len(tt.wantToEmails)

			msgs := testEmailer.GetSentMessages()
			ts.Len(msgs, wantCount, "incorrect message count")

			gotTos := testEmailer.GetAllToAddresses()
			ts.Equal(tt.wantToEmails, gotTos)

			for i, w := range tt.wantSubjectsContain {
				ts.Contains(msgs[i].Subject, w, "incorrect email subject")
			}
		})
	}
}

func (ts *TestSuite) Test_claimPreapproved() {
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

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	receiptClaim := f.Claims[0]
	models.UpdateClaimStatus(db, receiptClaim, api.ClaimStatusReceipt)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name                string
		event               events.Event
		wantToEmails        []string
		wantSubjectsContain []string
	}{
		{
			name: "preapproved",
			event: events.Event{
				Kind:    domain.EventApiClaimPreapproved,
				Payload: getTestPayload(receiptClaim.ID, &testEmailer),
			},
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"receipts are needed on your new claim",
				"receipts are needed on your new claim",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()

			claimPreapproved(tt.event)

			wantCount := len(tt.wantToEmails)

			msgs := testEmailer.GetSentMessages()
			ts.Len(msgs, wantCount, "incorrect message count")

			gotTos := testEmailer.GetAllToAddresses()
			ts.Equal(tt.wantToEmails, gotTos)

			for i, w := range tt.wantSubjectsContain {
				ts.Contains(msgs[i].Subject, w, "incorrect email subject")
			}
		})
	}
}

func (ts *TestSuite) Test_claimReceipt() {
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

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	receiptClaim := f.Claims[0]
	models.UpdateClaimStatus(db, receiptClaim, api.ClaimStatusReceipt)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name                string
		event               events.Event
		wantToEmails        []string
		wantSubjectsContain []string
	}{
		{
			name: "receipt required",
			event: events.Event{
				Kind:    domain.EventApiClaimReceipt,
				Payload: getTestPayload(receiptClaim.ID, &testEmailer),
			},
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"new receipts are needed on your claim",
				"new receipts are needed on your claim",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()

			claimReceipt(tt.event)

			wantCount := len(tt.wantToEmails)

			msgs := testEmailer.GetSentMessages()
			ts.Len(msgs, wantCount, "incorrect message count")

			gotTos := testEmailer.GetAllToAddresses()
			ts.Equal(tt.wantToEmails, gotTos)

			for i, w := range tt.wantSubjectsContain {
				ts.Contains(msgs[i].Subject, w, "incorrect email subject")
			}
		})
	}
}
