package api

import (
	"testing"

	"github.com/silinternational/cover-api/domain"
)

func (ts *TestSuite) TestGetPayoutOptionDescription() {
	domain.Env.RepairThresholdString = "70%"

	tests := []struct {
		name              string
		option            PayoutOption
		minimumDeductible Currency
		deductibleRate    float64
		want              string
	}{
		{
			name:              "fixed fraction",
			option:            PayoutOptionFixedFraction,
			minimumDeductible: 5,
			deductibleRate:    .03,
			want:              "Payout is a fixed portion of the item's covered value.",
		},
		{
			name:              "repair, no minimum",
			option:            PayoutOptionRepair,
			minimumDeductible: 0,
			deductibleRate:    .03,
			want:              "Payout is the item's covered value, the repair cost, or 70% of the item's fair market value, whichever is less, minus a 3% deductible.",
		},
		{
			name:              "repair, with minimum",
			option:            PayoutOptionRepair,
			minimumDeductible: 100,
			deductibleRate:    .03,
			want:              "Payout is the item's covered value, the repair cost, or 70% of the item's fair market value, whichever is less, minus a 3% deductible, subject to a minimum deductible of $1.00.",
		},
		{
			name:              "replacement, no minimum",
			option:            PayoutOptionReplacement,
			minimumDeductible: 0,
			deductibleRate:    .03,
			want:              "Payout is the item's covered value or the replacement cost, whichever is less, minus a 3% deductible.",
		},
		{
			name:              "replacement, with minimum",
			option:            PayoutOptionReplacement,
			minimumDeductible: 100,
			deductibleRate:    .03,
			want:              "Payout is the item's covered value or the replacement cost, whichever is less, minus a 3% deductible, subject to a minimum deductible of $1.00.",
		},
		{
			name:              "fmv, no minimum",
			option:            PayoutOptionFMV,
			minimumDeductible: 0,
			deductibleRate:    .03,
			want:              "Payout is the item's fair market value minus a 3% deductible.",
		},
		{
			name:              "fmv, with minimum",
			option:            PayoutOptionFMV,
			minimumDeductible: 100,
			deductibleRate:    .03,
			want:              "Payout is the item's fair market value minus a 3% deductible, subject to a minimum deductible of $1.00.",
		},
	}
	for _, tt := range tests {
		ts.T().Run(tt.name, func(t *testing.T) {
			got := GetPayoutOptionDescription(tt.option, tt.minimumDeductible, tt.deductibleRate)
			ts.Equal(tt.want, got)
		})
	}
}
