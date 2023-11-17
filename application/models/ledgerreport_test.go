package models

import (
	"testing"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/fin"
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
					ContentType: domain.ContentCSV,
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
					ContentType: domain.ContentCSV,
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
					ContentType: domain.ContentCSV,
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
		name            string
		date            time.Time
		reportFormat    string
		reportType      string
		want            LedgerReport
		wantContentType string
		wantErr         *api.AppError
	}{
		{
			name:         "invalid report type",
			date:         may,
			reportType:   "invalid",
			reportFormat: fin.ReportFormatSage,
			wantErr:      &api.AppError{Key: api.ErrorInvalidReportType, Category: api.CategoryUser},
		},
		{
			name:         "none found",
			date:         april,
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatSage,
			wantErr:      &api.AppError{Key: api.ErrorNoLedgerEntries, Category: api.CategoryNotFound},
		},
		{
			name:         "future date",
			date:         time.Now().UTC().AddDate(0, 0, 1),
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatSage,
			wantErr:      &api.AppError{Key: api.ErrorInvalidDate, Category: api.CategoryUser},
		},
		{
			name:         "one entry, sage",
			date:         may,
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatSage,
			want: LedgerReport{
				Type:          ReportTypeMonthly,
				Date:          may,
				File:          File{},
				LedgerEntries: nil,
			},
			wantContentType: domain.ContentCSV,
		},
		{
			name:         "one entry, netsuite",
			date:         may,
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatNetSuite,
			want: LedgerReport{
				Type:          ReportTypeMonthly,
				Date:          may,
				File:          File{},
				LedgerEntries: nil,
			},
			wantContentType: domain.ContentCSV,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got, err := NewLedgerReport(ctx, tt.reportFormat, tt.reportType, tt.date)
			if tt.wantErr != nil {
				ms.Error(err, "test should have produced an error")
				ms.EqualAppError(*tt.wantErr, err)
				return
			}

			ms.NoError(err)

			ms.Equal(tt.want.Type, got.Type, "incorrect report Type")
			ms.Equal(tt.want.Date, got.Date, "incorrect report Date")
			ms.Equal(tt.wantContentType, got.File.ContentType, "incorrect ContentType")
			ms.Equal(user.ID, got.File.CreatedByID, "incorrect CreatedByID")
			ms.Equal(1, len(got.LedgerEntries), "incorrect number of LedgerEntries")
		})
	}
}

func (ms *ModelSuite) TestNewPolicyLedgerReport() {
	// create ledger entries for a different policy to ensure they're not included in the results
	f0 := CreateLedgerFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})

	f := CreateLedgerFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})
	user := f.Users[0]
	policy := f.Policies[0]
	ctx := CreateTestContext(user)

	now := time.Now().UTC()
	nextMonth := now.AddDate(0, 1, 0)

	january := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	march := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	april := time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC)
	may := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)

	datesSubmitted := []time.Time{january, march, april}
	datesEntered := []nulls.Time{nulls.NewTime(march), nulls.NewTime(april), {}}
	entries := f.LedgerEntries
	others := f0.LedgerEntries // These should not get included in the results
	for i := range entries {
		entries[i].DateSubmitted = datesSubmitted[i]
		entries[i].DateEntered = datesEntered[i]
		Must(ms.DB.Update(&entries[i]))

		others[i].DateSubmitted = datesSubmitted[i]
		others[i].DateEntered = datesEntered[i]
		Must(ms.DB.Update(&others[i]))
	}

	tests := []struct {
		name         string
		date         time.Time
		month        int
		year         int
		reportType   string
		reportFormat string
		want         LedgerReport
		wantCount    int
		wantErr      *api.AppError
	}{
		{
			name:         "invalid report type",
			month:        int(may.Month()),
			year:         may.Year(),
			reportType:   "invalid",
			reportFormat: fin.ReportFormatSage,
			wantErr:      &api.AppError{Key: api.ErrorInvalidReportType, Category: api.CategoryUser},
		},
		{
			name:         "invalid future month",
			month:        int(nextMonth.Month()),
			year:         nextMonth.Year(),
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatSage,
			wantErr:      &api.AppError{Key: api.ErrorInvalidDate, Category: api.CategoryUser},
		},
		{
			name:         "invalid report month",
			month:        0,
			year:         2020,
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatSage,
			wantErr:      &api.AppError{Key: api.ErrorInvalidDate, Category: api.CategoryUser},
		},
		{
			name:         "invalid future year",
			month:        1,
			year:         now.Year() + 1,
			reportType:   ReportTypeAnnual,
			reportFormat: fin.ReportFormatSage,
			wantErr:      &api.AppError{Key: api.ErrorInvalidDate, Category: api.CategoryUser},
		},
		{
			name:         "one monthly entry",
			month:        int(april.Month()),
			year:         april.Year(),
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatSage,
			want: LedgerReport{
				Type:          ReportTypeMonthly,
				Date:          april,
				File:          File{},
				LedgerEntries: nil,
			},
			wantCount: 1,
		},
		{
			name:         "two annual entries",
			month:        int(may.Month()),
			year:         may.Year(),
			reportType:   ReportTypeAnnual,
			reportFormat: fin.ReportFormatSage,
			want: LedgerReport{
				Type:          ReportTypeAnnual,
				Date:          january,
				File:          File{},
				LedgerEntries: nil,
			},
			wantCount: 2,
		},
		{
			name:         "none found",
			month:        int(may.Month()),
			year:         may.Year(),
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatSage,
			want:         LedgerReport{},
			wantCount:    0,
		},
		{
			name:         "one monthly entry",
			month:        int(april.Month()),
			year:         april.Year(),
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatNetSuite,
			want: LedgerReport{
				Type:          ReportTypeMonthly,
				Date:          april,
				File:          File{},
				LedgerEntries: nil,
			},
			wantCount: 1,
		},
		{
			name:         "two annual entries",
			month:        int(may.Month()),
			year:         may.Year(),
			reportType:   ReportTypeAnnual,
			reportFormat: fin.ReportFormatNetSuite,
			want: LedgerReport{
				Type:          ReportTypeAnnual,
				Date:          january,
				File:          File{},
				LedgerEntries: nil,
			},
			wantCount: 2,
		},
		{
			name:         "none found",
			month:        int(may.Month()),
			year:         may.Year(),
			reportType:   ReportTypeMonthly,
			reportFormat: fin.ReportFormatNetSuite,
			want:         LedgerReport{},
			wantCount:    0,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got, err := NewPolicyLedgerReport(ctx, policy, tt.reportType, tt.month, tt.year)
			if tt.wantErr != nil {
				ms.Error(err, "test should have produced an error")
				ms.EqualAppError(*tt.wantErr, err)
				return
			}

			ms.NoError(err)
			ms.Equal(tt.wantCount, len(got.LedgerEntries), "incorrect number of LedgerEntries")

			if tt.wantCount == 0 {
				ms.Equal(uuid.Nil, got.PolicyID.UUID, "incorrect report PolicyID")
				return
			}

			ms.Equal(tt.want.Type, got.Type, "incorrect report Type")
			ms.Equal(policy.ID, got.PolicyID.UUID, "incorrect report PolicyID")
			ms.Equal(tt.want.Date, got.Date, "incorrect report Date")
			ms.Equal(domain.ContentCSV, got.File.ContentType, "incorrect ContentType")
			ms.Equal(user.ID, got.File.CreatedByID, "incorrect CreatedByID")
		})
	}
}

func (ms *ModelSuite) TestLedgerReport_AllNonPolicy() {
	f := CreateLedgerFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})
	leFixtures := f.LedgerEntries
	user := f.Users[0]
	policy := f.Policies[0]
	now := time.Now().UTC()

	ff := CreateFileFixtures(ms.DB, 3, user.ID)

	reports := LedgerReports{
		LedgerReport{ // No policy_id
			Type:          ReportTypeAnnual,
			Date:          now,
			File:          ff.Files[0],
			LedgerEntries: LedgerEntries{leFixtures[0]},
		},
		{ // Has a policy_id
			Type:          ReportTypeMonthly,
			Date:          now,
			File:          ff.Files[1],
			LedgerEntries: LedgerEntries{leFixtures[0]},
			PolicyID:      nulls.NewUUID(policy.ID),
		},
		{ // No policy_id
			Type:          ReportTypeAnnual,
			Date:          now,
			File:          ff.Files[2],
			LedgerEntries: LedgerEntries{leFixtures[0]},
		},
	}

	CreateLedgerReportFixtures(ms.DB, &reports)

	tests := []struct {
		name string
		want LedgerReports
	}{
		{
			name: "two reports",
			want: LedgerReports{reports[0], reports[2]},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			var got LedgerReports
			ms.NoError(got.AllNonPolicy(ms.DB))
			ms.Len(got, len(tt.want), "incorrect number of LedgerReports")

			for i, w := range tt.want {
				ms.Equal(w.FileID, got[i].FileID, "incorrect report FileID")
				ms.False(got[i].PolicyID.Valid, "incorrect null policy_id")
			}
		})
	}
}

func (ms *ModelSuite) TestLedgerReport_AllForPolicy() {
	f := CreateLedgerFixtures(ms.DB, FixturesConfig{NumberOfPolicies: 3, ItemsPerPolicy: 3})
	leFixtures := f.LedgerEntries
	user := f.Users[0]
	policy0 := f.Policies[0] // index will correspond to number of associated ledger reports
	policy1 := f.Policies[1]
	policy2 := f.Policies[2]
	now := time.Now().UTC()

	ff := CreateFileFixtures(ms.DB, 4, user.ID)

	reports := LedgerReports{
		LedgerReport{ // No policy_id
			Type:          ReportTypeAnnual,
			Date:          now,
			File:          ff.Files[0],
			LedgerEntries: LedgerEntries{leFixtures[0]},
		},
		{ // For policy with one ledger report
			Type:          ReportTypeAnnual,
			Date:          now,
			File:          ff.Files[1],
			LedgerEntries: LedgerEntries{leFixtures[0]},
			PolicyID:      nulls.NewUUID(policy1.ID),
		},
		{ // For policy with two ledger reports
			Type:          ReportTypeMonthly,
			Date:          now,
			File:          ff.Files[2],
			LedgerEntries: LedgerEntries{leFixtures[0]},
			PolicyID:      nulls.NewUUID(policy2.ID),
		},
		{ // For policy with two ledger reports
			Type:          ReportTypeMonthly,
			Date:          now,
			File:          ff.Files[3],
			LedgerEntries: LedgerEntries{leFixtures[0]},
			PolicyID:      nulls.NewUUID(policy2.ID),
		},
	}
	CreateLedgerReportFixtures(ms.DB, &reports)

	tests := []struct {
		name   string
		policy Policy
		want   LedgerReports
	}{
		{
			name:   "no reports",
			policy: policy0,
			want:   LedgerReports{},
		},
		{
			name:   "one report",
			policy: policy1,
			want:   LedgerReports{reports[1]},
		},
		{
			name:   "two reports",
			policy: policy2,
			want:   LedgerReports{reports[2], reports[3]},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			var got LedgerReports
			ms.NoError(got.AllForPolicy(ms.DB, tt.policy.ID))
			ms.Len(got, len(tt.want), "incorrect number of LedgerReports")

			for i, w := range tt.want {
				ms.Equal(w.FileID, got[i].FileID, "incorrect report FileID")
				ms.Equal(tt.policy.ID, got[i].PolicyID.UUID, "incorrect policy_id")
			}
		})
	}
}

func (ms *ModelSuite) TestPolicyLedgerTable() {
	// create ledger entries for a different policy to ensure they're not included in the results
	f0 := CreateLedgerFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})

	f := CreateLedgerFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})
	user := f.Users[0]
	policy := f.Policies[0]
	premiumTotal := policy.calculateAnnualPremium(ms.DB)
	ms.Greaterf(int(premiumTotal), 0, "bad premiumTotal for test")

	ctx := CreateTestContext(user)

	now := time.Now().UTC()
	nextMonth := now.AddDate(0, 1, 0)

	january := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	march := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	april := time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC)
	may := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)

	datesSubmitted := []time.Time{january, march, april}
	datesEntered := []nulls.Time{nulls.NewTime(april), nulls.NewTime(april), {}}
	entries := f.LedgerEntries
	others := f0.LedgerEntries // These should not get included in the results
	netTransactionsApril := api.Currency(0)

	for i := range entries {
		entries[i].DateSubmitted = datesSubmitted[i]
		entries[i].DateEntered = datesEntered[i]
		Must(ms.DB.Update(&entries[i]))

		if datesEntered[i].Time.Month() == april.Month() {
			netTransactionsApril += entries[i].Amount
		}

		others[i].DateSubmitted = datesSubmitted[i]
		others[i].DateEntered = datesEntered[i]
		Must(ms.DB.Update(&others[i]))
	}

	ms.NotEqualf(netTransactionsApril, 0, "bad netTransactions for tests")

	type statuses struct{ statusBefore, statusAfter string }

	tests := []struct {
		name         string
		date         time.Time
		month        int
		year         int
		want         *api.LedgerTable
		wantStatuses []statuses
		wantErr      *api.AppError
	}{
		{
			name:    "invalid future month",
			month:   int(nextMonth.Month()),
			year:    nextMonth.Year(),
			wantErr: &api.AppError{Key: api.ErrorInvalidDate, Category: api.CategoryUser},
		},
		{
			name:    "invalid report month",
			month:   0,
			year:    2020,
			wantErr: &api.AppError{Key: api.ErrorInvalidDate, Category: api.CategoryUser},
		},
		{
			name:    "invalid future year",
			month:   1,
			year:    now.Year() + 1,
			wantErr: &api.AppError{Key: api.ErrorInvalidDate, Category: api.CategoryUser},
		},
		{
			name:  "two entries",
			month: int(april.Month()),
			year:  april.Year(),
			want: &api.LedgerTable{
				PayoutTotal:     0,
				PremiumTotal:    premiumTotal,
				PremiumRate:     domain.Env.PremiumFactor,
				NetTransactions: netTransactionsApril,
				ReportMonth:     int(april.Month()),
				ReportYear:      april.Year(),
			},
			wantStatuses: []statuses{
				{statusBefore: string(api.ItemCoverageStatusPending), statusAfter: string(api.ItemCoverageStatusApproved)},
				{statusBefore: string(api.ItemCoverageStatusPending), statusAfter: string(api.ItemCoverageStatusApproved)},
			},
		},
		{
			name:         "none found",
			month:        int(may.Month()),
			year:         may.Year(),
			wantStatuses: []statuses{},
		},
	}

	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got, err := PolicyLedgerTable(ctx, policy, tt.month, tt.year)
			if tt.wantErr != nil {
				ms.Error(err, "test should have produced an error")
				ms.EqualAppError(*tt.wantErr, err)
				return
			}

			ms.NoError(err)
			ms.Len(got.Entries, len(tt.wantStatuses), "incorrect number of LedgerEntries")

			if tt.want == nil {
				return
			}

			ms.Equal(tt.want.PayoutTotal, got.PayoutTotal, "incorrect PayoutTotal")
			ms.Equal(tt.want.PremiumTotal, got.PremiumTotal, "incorrect PremiumTotal")
			ms.Equal(tt.want.PremiumRate, got.PremiumRate, "incorrect PremiumRate")
			ms.Equal(tt.want.NetTransactions, got.NetTransactions, "incorrect NetTransactions")
			ms.Equal(tt.want.ReportMonth, got.ReportMonth, "incorrect ReportMonth")
			ms.Equal(tt.want.ReportYear, got.ReportYear, "incorrect ReportYear")

			for i, e := range got.Entries {
				ms.Equal(tt.wantStatuses[i].statusBefore, e.StatusBefore, "incorrect statusBefore")
				ms.Equal(tt.wantStatuses[i].statusAfter, e.StatusAfter, "incorrect statusAfter")
			}
		})
	}
}

func (ms *ModelSuite) Test_isSafeToRenewAnnual() {
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			name: "January",
			now:  time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "February",
			now:  time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "March",
			now:  time.Date(2020, 3, 31, 0, 0, 0, 0, time.UTC),
			want: false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := IsSafeToRenewAnnual(ms.DB, tt.now)
			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) Test_isSafeToRenewMonthly() {
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			name: "1st of the month",
			now:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "19th of the month",
			now:  time.Date(2020, 1, 19, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "20th of the month",
			now:  time.Date(2020, 1, 20, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "31st of the month",
			now:  time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC),
			want: true,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := IsSafeToRenewMonthly(ms.DB, tt.now)
			ms.Equal(tt.want, got)
		})
	}
}
