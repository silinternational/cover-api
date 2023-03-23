package job

import (
	"time"

	"github.com/gobuffalo/buffalo/worker"
	"github.com/gobuffalo/pop/v6"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
	"github.com/silinternational/cover-api/models"
)

// inactivateItemsHandler is the Worker handler for inactivating items that
// have a coverage end date in the past
func inactivateItemsHandler(_ worker.Args) error {
	defer resubmitInactivateJob()

	ctx := createJobContext()

	err := models.DB.Transaction(func(tx *pop.Connection) error {
		ctx.Set(domain.ContextKeyTx, tx)
		var items models.Items
		return items.InactivateApprovedButEnded(ctx)
	})

	return err
}

func resubmitInactivateJob() {
	// Run twice a day, in case it errors out
	delay := time.Hour * 12

	// uncomment this in development, if you want it to run more often for debugging
	// delay = time.Second * 10

	if err := SubmitDelayed(InactivateItems, delay, map[string]any{}); err != nil {
		log.Errorf("error resubmitting inactivateItemsHandler: " + err.Error())
	}
	return
}
