package fin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/silinternational/cover-api/domain"
)

func Test_getFiscalPeriod(t *testing.T) {
	domain.Env.FiscalStartMonth = 9

	tests := []struct {
		month int
		want  int
	}{
		{
			month: 1,
			want:  5,
		},
		{
			month: 2,
			want:  6,
		},
		{
			month: 3,
			want:  7,
		},
		{
			month: 4,
			want:  8,
		},
		{
			month: 5,
			want:  9,
		},
		{
			month: 6,
			want:  10,
		},
		{
			month: 7,
			want:  11,
		},
		{
			month: 8,
			want:  12,
		},
		{
			month: 9,
			want:  1,
		},
		{
			month: 10,
			want:  2,
		},
		{
			month: 11,
			want:  3,
		},
		{
			month: 12,
			want:  4,
		},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("month=%d", tt.month)
		t.Run(name, func(t *testing.T) {
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
		{
			name:        "February",
			fiscalStart: 9,
			date:        time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
			want:        2021,
		},
		{
			name:        "March",
			fiscalStart: 9,
			date:        time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC),
			want:        2021,
		},
		{
			name:        "April",
			fiscalStart: 9,
			date:        time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC),
			want:        2021,
		},
		{
			name:        "May",
			fiscalStart: 9,
			date:        time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
			want:        2021,
		},
		{
			name:        "June",
			fiscalStart: 9,
			date:        time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
			want:        2021,
		},
		{
			name:        "July",
			fiscalStart: 9,
			date:        time.Date(2021, 7, 1, 0, 0, 0, 0, time.UTC),
			want:        2021,
		},
		{
			name:        "October",
			fiscalStart: 9,
			date:        time.Date(2021, 10, 1, 0, 0, 0, 0, time.UTC),
			want:        2022,
		},
		{
			name:        "November",
			fiscalStart: 9,
			date:        time.Date(2021, 11, 1, 0, 0, 0, 0, time.UTC),
			want:        2022,
		},
		{
			name:        "December",
			fiscalStart: 9,
			date:        time.Date(2021, 12, 1, 0, 0, 0, 0, time.UTC),
			want:        2022,
		},
	}
	for _, tt := range tests {
		domain.Env.FiscalStartMonth = tt.fiscalStart

		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getFiscalYear(tt.date))
		})
	}
}
