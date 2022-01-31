package models

import (
	"testing"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestLedgerReport_Create() {
	f := CreateLedgerFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2})
	user := f.Users[0]
	leFixtures := f.LedgerEntries

	date1 := time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2022, 1, 29, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		ledgerReport LedgerReport
		wantErr      *api.AppError
	}{
		{
			name: "validation error, missing filename",
			ledgerReport: LedgerReport{
				Type: ReportTypeAnnual,
				Date: date1,
				File: File{
					ContentType: "text/csv",
					CreatedByID: user.ID,
					Content:     []byte("a,b\n1,2"),
				},
				LedgerEntries: LedgerEntries{leFixtures[0]},
			},
			wantErr: &api.AppError{Key: api.ErrorFilenameRequired, Category: api.CategoryUser},
		},
		{
			name: "one LedgerEntry",
			ledgerReport: LedgerReport{
				Type: ReportTypeAnnual,
				Date: date1,
				File: File{
					Name:        "report1.csv",
					ContentType: "text/csv",
					CreatedByID: user.ID,
					Content:     []byte("a,b\n1,2"),
				},
				LedgerEntries: LedgerEntries{leFixtures[0]},
			},
			wantErr: nil,
		},
		{
			name: "two LedgerEntries",
			ledgerReport: LedgerReport{
				Type: ReportTypeAnnual,
				Date: date2,
				File: File{
					Name:        "report2.csv",
					ContentType: "text/csv",
					CreatedByID: user.ID,
					Content:     []byte("a,b\n1,2"),
				},
				LedgerEntries: leFixtures,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			// pop.Debug = true
			err := tt.ledgerReport.Create(ms.DB)
			if tt.wantErr != nil {
				ms.Error(err, "test should have produced an error")
				ms.EqualAppError(*tt.wantErr, err)
				return
			}

			ms.NoError(err)

			var lr LedgerReport
			err = ms.DB.Where("date = ?", tt.ledgerReport.Date).Eager().First(&lr)
			ms.NoError(err, "no report created")
			ms.Len(lr.LedgerEntries, len(tt.ledgerReport.LedgerEntries), "wrong number of ledger entries")
			ms.False(lr.File.ID == uuid.Nil, "file wasn't created")
			ms.Equal(tt.ledgerReport.File.Name, lr.File.Name, "incorrect filename")
			ms.True(lr.File.Linked, "incorrect Linked")
		})
	}
}

func (ms *ModelSuite) TestLedgerReport_ConvertToAPI() {
	id := domain.GetUUID()
	user := CreateUserFixtures(ms.DB, 1).Users[0]
	fileID := CreateFileFixtures(ms.DB, 1, user.ID).Files[0].ID
	date := time.Date(2022, 1, 28, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Now()
	createdAt := updatedAt.Add(-1 * time.Hour)
	c := &LedgerReport{
		ID:        id,
		FileID:    fileID,
		Type:      ReportTypeMonthly,
		Date:      date,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	got := c.ConvertToAPI(ms.DB)

	ms.Equal(id, got.ID, "ID is incorrect")
	ms.Equal(fileID, got.File.ID, "File ID is incorrect")
	ms.Equal(c.Type, got.Type, "Type is incorrect")
	ms.Equal(c.Date, got.Date, "Date is incorrect")
	ms.Equal(createdAt, got.CreatedAt, "CreatedAt is incorrect")
	ms.Equal(updatedAt, got.UpdatedAt, "UpdatedAt is incorrect")

	// At least make sure the URL expiration is updated. The File.ConvertToAPI test should cover the rest.
	ms.WithinDuration(updatedAt.Add(time.Minute*10), got.File.URLExpiration, time.Minute*2)
}

func (ms *ModelSuite) TestNewLedgerReport() {
	f := CreateLedgerFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2})
	user := f.Users[0]
	ctx := CreateTestContext(user)

	march := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	april := time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC)
	may := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)

	datesSubmitted := []time.Time{march, april}
	datesEntered := []nulls.Time{nulls.NewTime(april), {}}
	entries := f.LedgerEntries
	for i := range entries {
		entries[i].DateSubmitted = datesSubmitted[i]
		entries[i].DateEntered = datesEntered[i]
		Must(ms.DB.Update(&entries[i]))
	}

	tests := []struct {
		name       string
		date       time.Time
		reportType string
		want       LedgerReport
		wantErr    *api.AppError
	}{
		{
			name:       "invalid report type",
			date:       may,
			reportType: "invalid",
			wantErr:    &api.AppError{Key: api.ErrorInvalidReportType, Category: api.CategoryUser},
		},
		{
			name:       "none found",
			date:       april,
			reportType: ReportTypeMonthly,
			wantErr:    &api.AppError{Key: api.ErrorNoLedgerEntries, Category: api.CategoryNotFound},
		},
		{
			name:       "future date",
			date:       time.Now().UTC().AddDate(0, 0, 1),
			reportType: ReportTypeMonthly,
			wantErr:    &api.AppError{Key: api.ErrorInvalidDate, Category: api.CategoryUser},
		},
		{
			name:       "one entry",
			date:       may,
			reportType: ReportTypeMonthly,
			want: LedgerReport{
				Type:          ReportTypeMonthly,
				Date:          may,
				File:          File{},
				LedgerEntries: nil,
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got, err := NewLedgerReport(ctx, tt.reportType, tt.date)
			if tt.wantErr != nil {
				ms.Error(err, "test should have produced an error")
				ms.EqualAppError(*tt.wantErr, err)
				return
			}

			ms.NoError(err)

			ms.Equal(tt.want.Type, got.Type, "incorrect report Type")
			ms.Equal(tt.want.Date, got.Date, "incorrect report Date")
			ms.Equal("text/csv", got.File.ContentType, "incorrect ContentType")
			ms.Equal(user.ID, got.File.CreatedByID, "incorrect CreatedByID")
			ms.Equal(1, len(got.LedgerEntries), "incorrect number of LedgerEntries")
		})
	}
}
