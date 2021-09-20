package messages

import (
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/models"
)

// PolicyUserInviteQueueMessage queues messages to an invited policy user
func PolicyUserInviteQueueMessage(tx *pop.Connection, invite models.PolicyUserInvite) {
	// TODO do stuff
}
