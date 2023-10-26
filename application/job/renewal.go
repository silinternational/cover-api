package job

import (
	"time"

	"github.com/gobuffalo/buffalo/worker"
	"github.com/gobuffalo/pop/v6"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// annualRenewalHandler is the Worker handler for processing annual policy renewal
func annualRenewalHandler(_ worker.Args) error {
	err := models.DB.Transaction(func(tx *pop.Connection) error {
		endOfYear := domain.EndOfYear(time.Now().UTC().Year())

		var policies models.Policies
		if err := policies.AllActive(tx); err != nil {
			return err
		}
		return policies.ProcessRenewals(tx, endOfYear, domain.BillingPeriodAnnual)
	})
	return err
}

// monthlyRenewalHandler is the Worker handler for processing monthly policy renewal
func monthlyRenewalHandler(_ worker.Args) error {
	err := models.DB.Transaction(func(tx *pop.Connection) error {
		now := time.Now().UTC()

		var policies models.Policies
		if err := policies.AllActive(tx); err != nil {
			return err
		}
		return policies.ProcessRenewals(tx, now, domain.BillingPeriodMonthly)
	})
	return err
}
