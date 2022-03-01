package api

// PolicyTotals contains calculated totals for a policy
// swagger:model
type PolicyTotals struct {
	Items  ItemTotals  `json:"items"`
	Claims ClaimTotals `json:"claims"`
}

// ItemTotals contains totals calculated from all items
type ItemTotals struct {
	// coverage amount (0.01 USD)
	CoverageAmount Currency `json:"coverage_amount"`

	// annual premium (0.01 USD)
	AnnualPremium Currency `json:"annual_premium"`
}

// ClaimTotals contains totals calculated from all claims
type ClaimTotals struct {
	OpenClaims   int      `json:"open_claims"`
	PayoutAmount Currency `json:"payout_amount"`
}
