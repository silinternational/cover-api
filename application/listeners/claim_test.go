package listeners

import (
	"testing"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func getClaimFixtures(db *pop.Connection) models.Fixtures {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    1,
		UsersPerPolicy:      2,
		ClaimsPerPolicy:     2,
		ClaimItemsPerClaim:  1,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	return models.CreateItemFixtures(db, fixConfig)
}

type emailTest struct {
	wantToEmails        []string
	wantSubjectsContain []string
}

func validateClaimEmails(ts *TestSuite, eTest emailTest, testEmailer notifications.DummyEmailService) {
	wantCount := len(eTest.wantToEmails)

	msgs := testEmailer.GetSentMessages()
	ts.Len(msgs, wantCount, "incorrect message count")

	gotTos := testEmailer.GetAllToAddresses()
	ts.Equal(eTest.wantToEmails, gotTos)

	for i, w := range eTest.wantSubjectsContain {
		ts.Contains(msgs[i].Subject, w, "incorrect email subject")
	}
}

func (ts *TestSuite) Test_claimReview1() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	steward := models.CreateAdminUser(db)

	review1Claim := f.Claims[0]
	models.UpdateClaimStatus(db, review1Claim, api.ClaimStatusReview1)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
		eTest emailTest
	}{
		{
			name: "submitted to review1",
			event: events.Event{
				Kind:    domain.EventApiClaimReview1,
				Payload: newTestPayload(review1Claim.ID, &testEmailer),
			},
			eTest: emailTest{
				wantToEmails:        []string{steward.EmailOfChoice()},
				wantSubjectsContain: []string{"just (re)submitted a claim for approval"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimReview1(tt.event)
			validateClaimEmails(ts, tt.eTest, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_claimRevision() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	revisionClaim := f.Claims[0]
	models.UpdateClaimStatus(db, revisionClaim, api.ClaimStatusRevision)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
		eTest emailTest
	}{
		{
			name: "revisions required",
			event: events.Event{
				Kind:    domain.EventApiClaimRevision,
				Payload: newTestPayload(revisionClaim.ID, &testEmailer),
			},
			eTest: emailTest{
				wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
				wantSubjectsContain: []string{
					"changes have been requested on your claim",
					"changes have been requested on your claim",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimRevision(tt.event)
			validateClaimEmails(ts, tt.eTest, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_claimPreapproved() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	receiptClaim := f.Claims[0]
	models.UpdateClaimStatus(db, receiptClaim, api.ClaimStatusReceipt)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
		eTest emailTest
	}{
		{
			name: "preapproved",
			event: events.Event{
				Kind:    domain.EventApiClaimPreapproved,
				Payload: newTestPayload(receiptClaim.ID, &testEmailer),
			},
			eTest: emailTest{
				wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
				wantSubjectsContain: []string{
					"receipts are needed on your new claim",
					"receipts are needed on your new claim",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimPreapproved(tt.event)
			validateClaimEmails(ts, tt.eTest, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_claimReceipt() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	receiptClaim := f.Claims[0]
	models.UpdateClaimStatus(db, receiptClaim, api.ClaimStatusReceipt)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
		eTest emailTest
	}{
		{
			name: "receipt required",
			event: events.Event{
				Kind:    domain.EventApiClaimReceipt,
				Payload: newTestPayload(receiptClaim.ID, &testEmailer),
			},
			eTest: emailTest{
				wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
				wantSubjectsContain: []string{
					"new receipts are needed on your claim",
					"new receipts are needed on your claim",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimReceipt(tt.event)
			validateClaimEmails(ts, tt.eTest, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_claimReview2() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	steward := models.CreateAdminUser(db)

	review2Claim := f.Claims[0]
	models.UpdateClaimStatus(db, review2Claim, api.ClaimStatusReview2)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
		eTest emailTest
	}{
		{
			name: "submitted to review2",
			event: events.Event{
				Kind:    domain.EventApiClaimReview2,
				Payload: newTestPayload(review2Claim.ID, &testEmailer),
			},
			eTest: emailTest{
				wantToEmails:        []string{steward.EmailOfChoice()},
				wantSubjectsContain: []string{"just resubmitted a claim for approval"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimReview2(tt.event)
			validateClaimEmails(ts, tt.eTest, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_claimReview3() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	steward := models.CreateAdminUser(db)

	review3Claim := f.Claims[0]
	models.UpdateClaimStatus(db, review3Claim, api.ClaimStatusReview3)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
		eTest emailTest
	}{
		{
			name: "submitted to review3",
			event: events.Event{
				Kind:    domain.EventApiClaimReview3,
				Payload: newTestPayload(review3Claim.ID, &testEmailer),
			},
			eTest: emailTest{
				wantToEmails:        []string{steward.EmailOfChoice()},
				wantSubjectsContain: []string{"has a claim waiting for your approval"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimReview3(tt.event)
			validateClaimEmails(ts, tt.eTest, testEmailer)
		})
	}
}
