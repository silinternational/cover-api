package fin

import (
	"bytes"
	"fmt"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

const (
	policyHeader                 = `"Amount","Description","Reference","Date Entered"` + "\n"
	policyTransactionRowTemplate = `%s,"%s","%s",%s` + "\n"
)

type Policy struct {
	Transactions []Transaction
}

func (p *Policy) AppendToBatch(_ string, t Transaction) {
	if t.Amount != 0 {
		p.Transactions = append(p.Transactions, t)
	}
}

func (p *Policy) ToCSV() []byte {
	var buf bytes.Buffer
	buf.Write([]byte(policyHeader))
	for i := range p.Transactions {
		buf.Write(p.transactionRow(i))
	}

	return buf.Bytes()
}

func (p *Policy) ToZip() []byte {
	return nil
}

func (p *Policy) getReference(t Transaction) string {
	if t.Reference != nil {
		return *t.Reference
	}

	// For household policies
	if t.PolicyType == api.PolicyTypeHousehold {
		ref := fmt.Sprintf("MC %s", t.HouseholdID)

		if t.Name == "" {
			return ref
		}

		return fmt.Sprintf("%s / %s", ref, t.Name)
	}

	// For non-household policies
	ref := fmt.Sprintf("%s %s%s", t.EntityCode, t.AccountNumber, t.CostCenter)

	if t.PolicyName == "" {
		return ref
	}

	return fmt.Sprintf("%s / %s", ref, t.PolicyName)
}

func (p *Policy) transactionRow(rowNumber int) []byte {
	t := p.Transactions[rowNumber]
	str := fmt.Sprintf(
		policyTransactionRowTemplate,
		api.Currency(-t.Amount).String(),
		t.Description,
		p.getReference(t),
		t.Date.Format(domain.DateFormat),
	)
	return []byte(str)
}
