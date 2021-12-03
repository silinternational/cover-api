package messages

import (
	"fmt"
	"html/template"

	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// UserWelcomeQueueMessage queues a welcome message to a new user
func UserWelcomeQueueMessage(tx *pop.Connection, user models.User) {
	m := newEmailMessageData()
	m["personFirstName"] = user.FirstName
	m["emailIntro"] = template.HTML(domain.Env.UserWelcomeEmailIntro) // #nosec G203
	m["previewText"] = fmt.Sprintf("Dear %s, %s", user.FirstName, domain.Env.UserWelcomeEmailPreviewText)
	m["emailEnding"] = domain.Env.UserWelcomeEmailEnding
	m.addStewardData(tx)

	m["policyName"] = ""
	m["householdID"] = ""

	if len(user.Policies) == 0 {
		tx.Load(&user, "Policies") // Ignore errors and continue
	}

	if len(user.Policies) > 0 {
		policy := user.Policies[0]
		m["policyName"] = policy.Name
		m["householdID"] = policy.HouseholdID.String
	}

	notn := models.Notification{
		Body:    m.renderHTML(MessageTemplateUserWelcome),
		Subject: fmt.Sprintf("Welcome to %s!", domain.Env.AppName),

		// TODO make these constants somewhere
		Event:         "User Welcome Notification",
		EventCategory: "UserWelcome",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new User Welcome Notification: " + err.Error())
	}

	notn.CreateNotificationUserForUser(tx, user)
}
