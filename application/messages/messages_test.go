package messages

import (
	"testing"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

// TestSuite establishes a test suite for domain tests
type TestSuite struct {
	suite.Suite
	*require.Assertions
	DB *pop.Connection
}

func (ts *TestSuite) SetupTest() {
	ts.Assertions = require.New(ts.T())
	models.DestroyAll()
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	ts := &TestSuite{}
	c, err := pop.Connect(domain.Env.GoEnv)
	if err == nil {
		ts.DB = c
	}
	suite.Run(t, ts)
}

type testData struct {
	name                string
	wantToEmails        []string
	wantSubjectsContain []string
}

// TODO when ready, delete the testData type and rename this as testData
type testDataNew struct {
	name                  string
	wantToEmails          []interface{}
	wantSubjectContains   string
	wantInappTextContains string
	wantBodyContains      []string
}

func validateEmails(ts *TestSuite, td testData, testEmailer notifications.DummyEmailService) {
	wantCount := len(td.wantToEmails)

	msgs := testEmailer.GetSentMessages()
	ts.Len(msgs, wantCount, "incorrect message count")

	gotTos := testEmailer.GetAllToAddresses()
	ts.Equal(td.wantToEmails, gotTos)

	for i, w := range td.wantSubjectsContain {
		ts.Contains(msgs[i].Subject, w, "incorrect email subject")
	}
}

func validateNotificationUsers(ts *TestSuite, tx *pop.Connection, td testDataNew) {
	var notnUsers models.NotificationUsers

	ts.NoError(tx.Where("email_address in (?)",
		td.wantToEmails...).All(&notnUsers))

	ts.Equal(len(td.wantToEmails), len(notnUsers), "incorrect count of NotificationUsers")
	for _, n := range notnUsers {
		n.Load(tx)
		notn := n.Notification
		ts.Contains(notn.Subject, td.wantSubjectContains, "incorrect notification subject")
		ts.Contains(notn.InappText, td.wantInappTextContains, "incorrect notification inapp text")

		for _, c := range td.wantBodyContains {
			ts.Contains(notn.Body, c, "incorrect notification body")
		}

		ts.WithinDuration(time.Now().UTC(), n.SendAfterUTC, time.Minute,
			"incorrect NotificationUser SendAfterUTC")
	}
}

func (ts *TestSuite) Test_SendQueuedNotifications() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	user := f.Policies[0].Members[0]

	// Create queued notifications for different scenarios
	//  (already sent, should be sent, not ready to be sent)

	type notnFixture struct {
		name      string
		sendAfter time.Time
		sentAt    nulls.Time
		notnUser  models.NotificationUser
	}

	// The database truncates the time in the fractions of seconds somehow
	alreadySentTime := time.Now().UTC().Add(-1 * domain.DurationWeek).Truncate(time.Second)

	notnFixtures := []notnFixture{
		notnFixture{
			name:      "AlreadySent",
			sendAfter: time.Now().UTC().Add(-1 * domain.DurationWeek * 2),
			sentAt:    nulls.NewTime(alreadySentTime),
		},
		notnFixture{
			name:      "ToSend",
			sendAfter: time.Now().UTC().Add(-1 * domain.DurationWeek),
		},
		notnFixture{
			name:      "NotReady",
			sendAfter: time.Now().UTC().Add(domain.DurationWeek),
		},
		notnFixture{
			name:      "EmailError",
			sendAfter: time.Now().UTC().Add(-1 * domain.DurationWeek),
		},
	}

	notnUserFs := make([]models.NotificationUser, len(notnFixtures))

	nuAlreadySent := &notnUserFs[0]
	nuToSend := &notnUserFs[1]
	nuNotReady := &notnUserFs[2]
	nuEmailError := &notnUserFs[3]

	for i, n := range notnFixtures {
		notn := models.Notification{
			Subject: n.name + " message",
			Body:    "Body of " + n.name + " message.",
		}
		if n.name == "EmailError" {
			notn.Body = "ERROR"
		}
		models.MustCreate(db, &notn)

		notnUserFs[i] = models.NotificationUser{
			NotificationID: notn.ID,
			UserID:         user.ID,
			EmailAddress:   user.EmailOfChoice(),
			SendAfterUTC:   n.sendAfter,
			SentAtUTC:      n.sentAt,
		}
		models.MustCreate(db, &notnUserFs[i])
	}

	testEmailer := &notifications.TestEmailService

	tests := []testData{
		{
			name:                "send one email",
			wantToEmails:        []string{user.EmailOfChoice()},
			wantSubjectsContain: []string{"ToSend"},
			// TODO test whether the username is in the greeting
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			SendQueuedNotifications(db)
			validateEmails(ts, tt, *testEmailer)

			ts.NoError(nuAlreadySent.FindByID(db, nuAlreadySent.ID),
				"error fetching nuAlreadySent from database")
			ts.Equal(nuAlreadySent.SentAtUTC.Time, alreadySentTime,
				"incorrect SentAtUTC value for AlreadySent")

			ts.NoError(nuToSend.FindByID(db, nuToSend.ID),
				"error fetching nuToSend from database")
			ts.True(nuToSend.SentAtUTC.Valid, "missing SentAtUTC value")
			ts.WithinDuration(time.Now().UTC(), nuToSend.SentAtUTC.Time, time.Minute,
				"incorrect SentAtUTC value")

			ts.NoError(nuNotReady.FindByID(db, nuNotReady.ID),
				"error fetching nuNotReady from database")
			ts.False(nuNotReady.SentAtUTC.Valid, "SentAtUTC value should not be set")

			ts.NoError(nuEmailError.FindByID(db, nuEmailError.ID),
				"error fetching nuEmailError from database")
			ts.False(nuEmailError.SentAtUTC.Valid, "SentAtUTC value should not be set")
			ts.WithinDuration(time.Now().UTC(), nuEmailError.SendAfterUTC, time.Minute,
				"incorrect SendAfterUTC value")
			ts.WithinDuration(time.Now().UTC(), nuEmailError.LastAttemptUTC.Time, time.Minute,
				"incorrect LastAttemptUTC value")
			ts.Equal(1, nuEmailError.SendAttemptCount, "incorrect SendAttemptCount")
		})
	}
}

func (ts *TestSuite) Test_NextAttemptTime() {
	t := ts.T()

	tests := []struct {
		name         string
		attemptCount int
		want         time.Time
	}{
		{
			name:         "no delay",
			attemptCount: 0,
			want:         time.Now().UTC(),
		},
		{
			name:         "short delay",
			attemptCount: 2,
			want:         time.Now().UTC().Add(4 * time.Minute),
		},
		{
			name:         "medium delay",
			attemptCount: 9,
			want:         time.Now().UTC().Add(81 * time.Minute),
		},
		{
			name:         "long delay",
			attemptCount: 12,
			want:         time.Now().UTC().Add(100 * time.Minute),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextAttemptTime(tt.attemptCount)

			ts.WithinDuration(tt.want, got, time.Minute, "incorrect time for next attempt")
		})
	}
}
