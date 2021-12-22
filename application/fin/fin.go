package fin

import (
	"fmt"
	"time"

	"github.com/silinternational/cover-api/domain"
)

const ProviderTypeSage = "sage"

type Transaction struct {
	Account     string
	Amount      int
	Description string
	Reference   string
	Date        time.Time
}

type Provider interface {
	AppendToBatch(Transaction)
	BatchToCSV() []byte
}

func NewBatch(providerType string, date time.Time) Provider {
	batchDesc := fmt.Sprintf("%s %s JE", date.Format("January 2006"), domain.Env.AppName)

	switch providerType {
	case ProviderTypeSage:
		return &Sage{
			Period:             getFiscalPeriod(int(date.Month())),
			Year:               getFiscalYear(date),
			JournalDescription: batchDesc,
			Transactions:       nil,
		}
	}
	panic("fin: invalid provider type")
}

func getFiscalPeriod(month int) int {
	return (month-domain.Env.FiscalStartMonth+12)%12 + 1
}

func getFiscalYear(date time.Time) int {
	month := date.Month()
	year := date.Year()
	if domain.Env.FiscalStartMonth != 1 && int(month) >= domain.Env.FiscalStartMonth {
		return year + 1
	}
	return year
}
