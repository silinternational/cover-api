package notifications

import (
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

type Message struct {
	Body      string
	Subject   string
	Template  string
	Data      map[string]interface{}
	FromName  string
	FromEmail string
	FromPhone string
	ToName    string
	ToEmail   string
	ToPhone   string
}

// NewEmailMessage returns a message with the FromEmail, the Data.appName and Data.uiURL already set
func NewEmailMessage() Message {
	msg := Message{
		FromEmail: domain.EmailFromAddress(nil),
		Data: map[string]interface{}{
			"appName": domain.Env.AppName,
			"uiURL":   domain.Env.UIURL,
		},
	}
	return msg
}

// CopyToStewards returns one message for each steward with
//   the ToName and ToEmail based on the steward's information
func (m Message) CopyToStewards() []Message {
	var stewards models.Users
	stewards.FindStewards(models.DB)

	msgs := make([]Message, len(stewards))
	for i, s := range stewards {
		newMsg := m
		newMsg.ToName = s.Name()
		newMsg.ToEmail = s.EmailOfChoice()
		msgs[i] = newMsg
	}

	return msgs
}

// CopyToSignators returns one message for each signator with
//   the ToName and ToEmail based on the signator's information
func (m Message) CopyToSignators() []Message {
	var signators models.Users
	signators.FindSignators(models.DB)

	msgs := make([]Message, len(signators))
	for i, s := range signators {
		newMsg := m
		newMsg.ToName = s.Name()
		newMsg.ToEmail = s.EmailOfChoice()
		msgs[i] = newMsg
	}

	return msgs
}
