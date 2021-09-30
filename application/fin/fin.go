package fin

import (
	"fmt"
	"time"

	"github.com/silinternational/cover-api/domain"
)

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
	case "sage":
		return &Sage{
			Period:             getFiscalPeriod(int(date.Month())),
			Year:               date.Year(),
			JournalDescription: batchDesc,
			Transactions:       nil,
		}
	}
	panic("fin: invalid provider type")
}

func getFiscalPeriod(month int) int {
	return (month-domain.Env.FiscalStartMonth+12)%12 + 1
}
