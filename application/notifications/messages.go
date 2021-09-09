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
	var stewards models.Users
	stewards.FindStewards(models.DB)

	m.ToName = stewards[0].Name()
	m.ToEmail = stewards[0].EmailOfChoice()
	return m
}

// sets the msg ToName and ToEmail based on the signator's information
func (m Message) AddToSignator() Message {
	var signators models.Users
	signators.FindSignators(models.DB)

	m.ToName = signators[0].Name()
	m.ToEmail = signators[0].EmailOfChoice()
	return m
}
