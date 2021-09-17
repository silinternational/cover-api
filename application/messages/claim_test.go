package messages

import (
	"testing"

	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
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

func (ts *TestSuite) Test_ClaimReview1QueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	steward := models.CreateAdminUsers(db)[models.AppRoleSteward]

	review1Claim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReview1)

	tests := []testData{
		{
			name:                  "submitted to review1",
			wantToEmails:          []interface{}{steward.EmailOfChoice()},
			wantSubjectContains:   "just (re)submitted a claim for approval",
			wantInappTextContains: "A new claim is waiting for your approval",
			wantBodyContains: []string{
				domain.Env.UIURL,
				review1Claim.ReferenceNumber,
				"just submitted a claim which needs your attention.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimReview1QueueMessage(db, review1Claim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimRevisionQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	revisionClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusRevision)

	tests := []testData{
		{
			name:                  "revisions requiredd",
			wantToEmails:          []interface{}{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "changes have been requested on your claim",
			wantInappTextContains: "changes have been requested on your new claim",
			wantBodyContains: []string{
				domain.Env.UIURL,
				revisionClaim.ReferenceNumber,
				"The claim you submitted has not yet been approved.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimRevisionQueueMessage(db, revisionClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}
func (ts *TestSuite) Test_ClaimPreapprovedQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	receiptClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReceipt)

	tests := []testData{
		{
			name:                  "preapproved",
			wantToEmails:          []interface{}{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "receipt(s) needed on your new claim",
			wantInappTextContains: "receipts are needed on your new claim",
			wantBodyContains: []string{
				domain.Env.UIURL,
				receiptClaim.ReferenceNumber,
				"The claim you submitted has been tentatively preapproved.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimPreapprovedQueueMessage(db, receiptClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimReceiptQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	receiptClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReceipt)

	tests := []testData{
		{
			name:                  "receipts required again",
			wantToEmails:          []interface{}{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "new receipt(s) needed on your claim",
			wantInappTextContains: "new/different receipts are needed on your claim",
			wantBodyContains: []string{
				domain.Env.UIURL,
				receiptClaim.ReferenceNumber,
				"The claim you submitted still has some receipts that are needed.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimReceiptQueueMessage(db, receiptClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimReview2QueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	steward := models.CreateAdminUsers(db)[models.AppRoleSteward]

	review2Claim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReview2)

	tests := []testData{
		{
			name:                  "submitted to review2",
			wantToEmails:          []interface{}{steward.EmailOfChoice()},
			wantSubjectContains:   "just resubmitted a claim for approval",
			wantInappTextContains: "A claim is waiting for your approval",
			wantBodyContains: []string{
				domain.Env.UIURL,
				review2Claim.ReferenceNumber,
				"has resubmitted a claim which needs your attention.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimReview2QueueMessage(db, review2Claim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimReview3QueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	signator := models.CreateAdminUsers(db)[models.AppRoleSignator]

	review3Claim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReview3)

	tests := []testData{
		{
			name:                  "submitted to review3",
			wantToEmails:          []interface{}{signator.EmailOfChoice()},
			wantSubjectContains:   "has a claim waiting for your approval",
			wantInappTextContains: "A claim is waiting for your approval",
			wantBodyContains: []string{
				domain.Env.UIURL,
				review3Claim.ReferenceNumber,
				"has submitted a claim which needs your attention.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimReview3QueueMessage(db, review3Claim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimApprovedQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	approvedClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusApproved)

	tests := []testData{
		{
			name:                  "coverage approved",
			wantToEmails:          []interface{}{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "your claim has been approved",
			wantInappTextContains: "your claim has been approved",
			wantBodyContains: []string{
				domain.Env.UIURL,
				approvedClaim.ReferenceNumber,
				"The claim you submitted has been approved.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimApprovedQueueMessage(db, approvedClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimDeniedQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	deniedClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusDenied)

	tests := []testData{
		{
			name:                  "coverage denied",
			wantToEmails:          []interface{}{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "your claim has been denied",
			wantInappTextContains: "your claim has been denied",
			wantBodyContains: []string{
				domain.Env.UIURL,
				deniedClaim.ReferenceNumber,
				"The claim you submitted has been denied.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimDeniedQueueMessage(db, deniedClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}
