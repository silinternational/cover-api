package fin

import (
	"fmt"
	"time"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

const (
	ReportFormatNetSuite = "netsuite"
	ReportFormatPolicy   = "policy"
	ReportFormatSage     = "sage"
)

type (
	Transactions      []Transaction
	TransactionBlocks map[string]Transactions // keyed by account
)

type Transaction struct {
	EntityCode        string
	RiskCategoryName  string
	RiskCategoryCC    string // Risk Category Cost Center
	Type              string
	PolicyType        api.PolicyType
	HouseholdID       string
	CostCenter        string
	AccountNumber     string
	IncomeAccount     string
	Name              string
	PolicyName        string
	ClaimPayoutOption string

	Account     string
	Amount      api.Currency
	Description string
	Reference   *string // Override the reference if given
	Date        time.Time
}

type Report interface {
	AppendToBatch(string, Transaction)
	RenderBatch() ([]byte, string)

	getReference(Transaction) string
}

func NewBatch(reportFormat, reportType string, date time.Time) Report {
	batchDesc := fmt.Sprintf("%s %s JE", date.Format("January 2006"), domain.Env.AppName)

	switch reportFormat {
	case ReportFormatNetSuite:
		return newNetSuiteReport(batchDesc, reportType, date)
	case ReportFormatPolicy:
		return &Policy{}
	case ReportFormatSage:
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
