package notifications

import (
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

type Message struct {
	Template  string
	Data      map[string]interface{}
	FromName  string
	FromEmail string
	FromPhone string
	ToName    string
	ToEmail   string
	ToPhone   string
	Subject   string
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

// sets the msg ToName and ToEmail based on the steward's information
func (m Message) AddToSteward() Message {
	var steward models.User
	steward.FindSteward(models.DB)

	m.ToName = steward.Name()
	m.ToEmail = steward.EmailOfChoice()
	return m
}
