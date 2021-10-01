package models

import (
	"testing"
	"time"

	"github.com/gobuffalo/nulls"
)

func (ms *ModelSuite) TestLedgerEntries_AllForMonth() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2})

	march := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	april := time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC)
	may := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)

	datesSubmitted := []time.Time{march, april}
	datesEntered := []nulls.Time{nulls.NewTime(april), {}}

	for i := range f.Items {
		ms.NoError(f.Items[i].Approve(ms.DB))

		entry := LedgerEntry{}
		ms.NoError(ms.DB.Where("item_id = ?", f.Items[i].ID).First(&entry))
		entry.DateSubmitted = datesSubmitted[i]
		entry.DateEntered = datesEntered[i]
		ms.NoError(ms.DB.Update(&entry))
	}

	tests := []struct {
		name                    string
		batchDate               time.Time
		expectedNumberOfEntries int
		wantErr                 bool
	}{
		{
			name:                    "no un-entered entries for March",
			batchDate:               march,
			expectedNumberOfEntries: 0,
			wantErr:                 false,
		},
		{
			name:                    "one entry for April",
			batchDate:               april,
			expectedNumberOfEntries: 1,
			wantErr:                 false,
		},
		{
			name:                    "no entry for May",
			batchDate:               may,
			expectedNumberOfEntries: 0,
			wantErr:                 false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			entries := LedgerEntries{}
			err := entries.AllForMonth(ms.DB, tt.batchDate)
			ms.NoError(err)
			ms.Equal(tt.expectedNumberOfEntries, len(entries), "incorrect number of LedgerEntries")
		})
	}
}
