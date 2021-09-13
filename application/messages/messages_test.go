package messages

import (
	"testing"
	"time"

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
	wantToEmails          []string
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
		td.wantToEmails[0], td.wantToEmails[1]).All(&notnUsers))

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
