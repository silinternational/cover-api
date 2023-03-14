package job

import (
	"time"

	"github.com/gobuffalo/buffalo/worker"
	"github.com/gobuffalo/pop/v6"
	"github.com/silinternational/cover-api/models"
)

// annualRenewalHandler is the Worker handler for processing annual policy renewal
func annualRenewalHandler(_ worker.Args) error {
	err := models.DB.Transaction(func(tx *pop.Connection) error {
		currentYear := time.Now().UTC().Year()

		var policies models.Policies
		if err := policies.AllActive(tx); err != nil {
			return err
		}
		return policies.ProcessAnnualCoverage(tx, currentYear)
	})
	return err
}
