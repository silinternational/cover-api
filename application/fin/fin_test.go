package fin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/silinternational/cover-api/domain"
)

func Test_getFiscalPeriod(t *testing.T) {
	domain.Env.FiscalStartMonth = 9

	tests := []struct {
		name  string
		month int
		want  int
	}{
		{
			name:  "September",
			month: 9,
			want:  1,
		},
		{
			name:  "August",
			month: 8,
			want:  12,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getFiscalPeriod(tt.month))
		})
	}
}

func Test_getFiscalYear(t *testing.T) {
	tests := []struct {
		name        string
		fiscalStart int
		date        time.Time
		want        int
	}{
		{
			name:        "first day of fiscal year",
			fiscalStart: 9,
			date:        time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC),
			want:        2022,
		},
		{
			name:        "last day of fiscal year",
			fiscalStart: 9,
			date:        time.Date(2021, 8, 31, 0, 0, 0, 0, time.UTC),
			want:        2021,
		},
		{
			name:        "fiscal start in January",
			fiscalStart: 1,
			date:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			want:        2021,
		},
	}
	for _, tt := range tests {
		domain.Env.FiscalStartMonth = tt.fiscalStart

		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getFiscalYear(tt.date))
		})
	}
}
