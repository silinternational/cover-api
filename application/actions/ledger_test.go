package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/fin"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_LedgerReportList() {
	f := as.createFixturesForLedger()
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	lr, err := models.NewLedgerReport(models.CreateTestContext(stewardUser), fin.ReportFormatSage, models.ReportTypeMonthly, time.Now())
	as.NoError(err)
	as.NoError(lr.Create(as.DB))

	tests := []struct {
		name        string
		actor       models.User
		wantReports int
		wantStatus  int
		wantInBody  []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:        "ok",
			actor:       stewardUser,
			wantStatus:  http.StatusOK,
			wantReports: 1,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(ledgerReportPath)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}

			if res.Code != http.StatusOK {
				return
			}

			var reports []api.LedgerReport
			as.NoError(json.Unmarshal([]byte(body), &reports))
			as.Equal(tt.wantReports, len(reports))
		})
	}
}

func (as *ActionSuite) Test_LedgerReportView() {
	otherUser := models.CreateUserFixtures(as.DB, 1).Users[0]

	f := as.createFixturesForLedger()
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]
	policy := f.Policies[0]

	now := time.Now().UTC()

	lr, err := models.NewLedgerReport(models.CreateTestContext(stewardUser), fin.ReportFormatSage, models.ReportTypeMonthly, now)
	as.NoError(err)
	as.NoError(lr.Create(as.DB))

	policyReport, err := models.NewPolicyLedgerReport(models.CreateTestContext(normalUser),
		policy, models.ReportTypeAnnual, 0, now.Year())
	as.NoError(err)
	as.NoError(policyReport.Create(as.DB))

	tests := []struct {
		name       string
		actor      models.User
		lrID       uuid.UUID
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			lrID:       lr.ID,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "not admin",
			actor:      normalUser,
			lrID:       lr.ID,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "not policy member",
			actor:      otherUser,
			lrID:       policyReport.ID,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "ok normalUser's own",
			actor:      normalUser,
			lrID:       policyReport.ID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "ok steward",
			actor:      stewardUser,
			lrID:       lr.ID,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(fmt.Sprintf("%s/%s", ledgerReportPath, tt.lrID))
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}

			if res.Code != http.StatusOK {
				return
			}

			var report api.LedgerReport
			as.NoError(json.Unmarshal([]byte(body), &report))
			as.Equal(tt.lrID, report.ID)
		})
	}
}

func (as *ActionSuite) Test_LedgerReportCreate() {
	f := as.createFixturesForLedger()
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name       string
		actor      models.User
		reportType string
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "invalid report type",
			actor:      stewardUser,
			reportType: "not-a-real-report-type",
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{`"key":"` + api.ErrorInvalidReportType.String()},
		},
		{
			name:       "monthly report",
			actor:      stewardUser,
			reportType: models.ReportTypeMonthly,
			wantStatus: http.StatusOK,
		},
		{
			name:       "annual report",
			actor:      stewardUser,
			reportType: models.ReportTypeAnnual,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(ledgerReportPath)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Post(api.LedgerReportCreateInput{
				Type: tt.reportType,
				Date: time.Now().UTC().Format(domain.DateFormat),
			})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}

			if res.Code != http.StatusOK {
				return
			}

			var report api.LedgerReport
			as.NoError(json.Unmarshal([]byte(body), &report))
			as.Equal(tt.reportType, report.Type)
		})
	}
}

func (as *ActionSuite) Test_LedgerReportReconcile() {
	f := as.createFixturesForLedger()
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	lr, err := models.NewLedgerReport(models.CreateTestContext(stewardUser), fin.ReportFormatSage, models.ReportTypeMonthly, time.Now())
	as.NoError(err)
	as.NoError(lr.Create(as.DB))

	tests := []struct {
		name       string
		actor      models.User
		date       string
		want       int // approved records
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "normal user nothing reconciled",
			actor:      stewardUser,
			date:       time.Now().AddDate(0, -1, 0).Format(domain.DateFormat),
			wantStatus: http.StatusOK,
			want:       0,
		},
		{
			name:       "steward user good results",
			actor:      stewardUser,
			date:       time.Now().Format(domain.DateFormat),
			wantStatus: http.StatusOK,
			want:       1,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(fmt.Sprintf("%s/%s", ledgerReportPath, lr.ID))
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Put(nil)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}

			if res.Code != http.StatusOK {
				return
			}

			var report api.LedgerReport
			as.NoError(json.Unmarshal([]byte(body), &report))
			as.Equal(lr.ID, report.ID)

			var le models.LedgerEntries
			as.NoError(as.DB.Where("item_id = ?", f.Items[1].ID).All(&le))
			as.Equal(1, len(le), "something is not right with the test setup")
			for i := range le {
				as.True(le[i].DateEntered.Valid, "ledger entry DateEntered was not set")
			}
		})
	}
}

func (as *ActionSuite) Test_LedgerAnnualProcess() {
	year := time.Now().UTC().Year()

	f := models.CreateItemFixtures(as.DB, models.FixturesConfig{ItemsPerPolicy: 3})

	f.Items[0].PaidThroughDate = domain.EndOfYear(year)
	models.UpdateItemStatus(as.DB, f.Items[0], api.ItemCoverageStatusApproved, "")
	models.UpdateItemStatus(as.DB, f.Items[1], api.ItemCoverageStatusApproved, "")

	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name       string
		actor      models.User
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "steward user good results",
			actor:      stewardUser,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(ledgerReportPath + "/annual")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Post(nil)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}
		})
	}
}

func (as *ActionSuite) Test_LedgerAnnualStatus() {
	year := time.Now().UTC().Year()

	f := models.CreateItemFixtures(as.DB, models.FixturesConfig{ItemsPerPolicy: 3})

	f.Items[0].PaidThroughDate = domain.EndOfYear(year)
	models.UpdateItemStatus(as.DB, f.Items[0], api.ItemCoverageStatusApproved, "")
	models.UpdateItemStatus(as.DB, f.Items[1], api.ItemCoverageStatusApproved, "")

	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name       string
		actor      models.User
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "steward user good results",
			actor:      stewardUser,
			wantStatus: http.StatusOK,
			wantInBody: []string{`"is_complete":false`, `"items_to_process":1`},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(ledgerReportPath + "/annual")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}
		})
	}
}

func (as *ActionSuite) createFixturesForLedger() models.Fixtures {
	f := models.CreateItemFixtures(as.DB, models.FixturesConfig{ItemsPerPolicy: 3})

	now := time.Now().UTC()
	yesterday := now.AddDate(0, 0, -1)

	datesSubmitted := []time.Time{yesterday, yesterday}
	datesEntered := []nulls.Time{nulls.NewTime(now), {}}

	for i := range datesSubmitted {
		f.Items[i].LoadPolicy(as.DB, false)
		f.Items[i].Policy.LoadMembers(as.DB, false)
		user := f.Items[i].Policy.Members[0]
		ctx := models.CreateTestContext(user)
		as.NoError(f.Items[i].Approve(ctx, false))

		entry := models.LedgerEntry{}
		as.NoError(as.DB.Where("item_id = ?", f.Items[i].ID).First(&entry))
		entry.DateSubmitted = datesSubmitted[i]
		entry.DateEntered = datesEntered[i]
		as.NoError(as.DB.Update(&entry))
	}

	// add an entry for the annual report
	as.NoError(f.Items[2].CreateLedgerEntry(as.DB, models.LedgerEntryTypeCoverageRenewal, 1000, now))
	as.NoError(f.Items[2].SetPaidThroughDate(as.DB, domain.EndOfYear(now.Year())))

	return f
}

func (as *ActionSuite) Test_LedgerMonthlyProcess() {
	now := time.Now().UTC()

	f := models.CreateItemFixtures(as.DB, models.FixturesConfig{ItemsPerPolicy: 3})

	f.Items[0].PaidThroughDate = domain.EndOfMonth(now)
	models.UpdateItemStatus(as.DB, f.Items[0], api.ItemCoverageStatusApproved, "")
	models.UpdateItemStatus(as.DB, f.Items[1], api.ItemCoverageStatusApproved, "")

	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name       string
		actor      models.User
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "steward user good results",
			actor:      stewardUser,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(ledgerReportPath + "/monthly")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Post(nil)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}
		})
	}
}

func (as *ActionSuite) Test_LedgerMonthlyStatus() {
	now := time.Now().UTC()

	f := models.CreateItemFixtures(as.DB, models.FixturesConfig{ItemsPerPolicy: 3})

	f.Items[0].PaidThroughDate = domain.EndOfMonth(now)
	f.ItemCategories[1].BillingPeriod = domain.BillingPeriodMonthly
	f.ItemCategories[1].RiskCategoryID = models.RiskCategoryVehicleID()
	models.UpdateItemStatus(as.DB, f.Items[0], api.ItemCoverageStatusApproved, "")
	models.UpdateItemStatus(as.DB, f.Items[1], api.ItemCoverageStatusApproved, "")
	as.NoError(as.DB.Update(&f.ItemCategories[1]))

	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name       string
		actor      models.User
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "steward user good results",
			actor:      stewardUser,
			wantStatus: http.StatusOK,
			wantInBody: []string{`"is_complete":false`, `"items_to_process":1`},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(ledgerReportPath + "/monthly")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}
		})
	}
}
