package notifications

import (
	"github.com/silinternational/cover-api/domain"
)

// TestSendraw can be used in a local environment for development. Add SES credentials to the appropriate
// environment variables, and change the "To" and "From" email addresses to valid addresses.
func (ts *TestSuite) TestSendRaw() {
	ts.T().Skip("only for use in local environment if configured with SES credentials")

	data := rawEmail(
		"me@example.com",
		domain.Env.EmailFromAddress,
		"test subject",
		`<h4>body</h4><img src="cid:logo"><p>End of body</p>`)

	err := SendRaw(domain.Env.EmailFromAddress, data)
	ts.NoError(err)
}
