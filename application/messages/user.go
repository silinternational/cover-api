package messages

import (
	"html/template"

	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// UserWelcomeQueueMessage queues a welcome message to a new user
func UserWelcomeQueueMessage(tx *pop.Connection, user models.User) {
	m := newEmailMessageData()
	m["uiURL"] = domain.Env.UIURL
	m["personFirstName"] = user.FirstName
	m["emailIntro"] = template.HTML(domain.Env.UserWelcomeEmailIntro) // #nosec G203
	m["previewText"] = domain.Env.UserWelcomeEmailPreviewText

	steward := models.GetDefaultSteward(tx)
	m["supportEmail"] = steward.Email
	m["supportName"] = steward.Name()
	m["supportFirstName"] = steward.FirstName

	notn := models.Notification{
		Body:    m.renderHTML(MessageTemplateUserWelcome),
		Subject: "Welcome to Cover!",

		// TODO make these constants somewhere
		Event:         "User Welcome Notification",
		EventCategory: "UserWelcome",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new User Welcome Notification: " + err.Error())
	}

	notn.CreateNotificationUserForUser(tx, user)
}
