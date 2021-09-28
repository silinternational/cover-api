package fin

import "time"

type Transaction struct {
	Account     string
	Amount      int
	Description string
	Reference   string
	Date        time.Time
}

type Provider interface {
	AppendToBatch(Transaction) error
	BatchToCSV() ([]byte, error)
}
