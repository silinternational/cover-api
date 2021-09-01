package notifications

import (
	"bytes"
	"os"

	"github.com/silinternational/cover-api/domain"
)

// TestSendEmail can be used in a local environment for development. Add SES credentials to the appropriate
// environment variables, and change the "To" and "From" email addresses to valid addresses.
func (ts *TestSuite) TestSendEmail() {
	ts.T().Skip("only for use in local environment if configured with SES credentials")
	err := SendEmail(
		"me@example.com",
		domain.Env.EmailFromAddress,
		"test subject",
		`<h4>body</h4><img src="cid:logo"><p>End of body</p>`)
	ts.NoError(err)
}

func (ts *TestSuite) TestRawEmail() {
	var buf bytes.Buffer
	domain.ErrLogger.SetOutput(&buf)

	defer domain.ErrLogger.SetOutput(os.Stderr)

	raw := rawEmail(
		"to@example.com",
		domain.Env.EmailFromAddress,
		"test subject",
		`<h4>body</h4><img src="cid:logo"><p>End of body</p>`)

	ts.Greater(len(raw), 1000)

	ts.Equal("", buf.String(), "Got an unexpected error log entry")
}
