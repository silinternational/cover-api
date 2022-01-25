package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_LedgerList() {
	f := as.createFixturesForLedger()
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

	tests := []struct {
		name       string
		actor      models.User
		reportType string
		format     string
		wantRows   int // rows in CSV, including header rows
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			reportType: reportTypeMonthly,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{`"key":"` + api.ErrorNotAuthorized.String()},
		},
		{
			name:       "insufficient privileges",
			actor:      normalUser,
			reportType: reportTypeMonthly,
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
			reportType: reportTypeMonthly,
			format:     "text/csv",
			wantStatus: http.StatusOK,
			wantRows:   5, // 2 header rows, 1 summary row, 1 transaction row, 1 balance row
		},
		{
			name:       "annual report",
			actor:      stewardUser,
			reportType: reportTypeAnnual,
			format:     "text/csv",
			wantStatus: http.StatusOK,
			wantRows:   5, // 2 header rows, 1 summary row, 1 transaction row, 1 balance row
		},
		{
			name:       "annual report, json",
			actor:      stewardUser,
			reportType: reportTypeAnnual,
			format:     "application/json",
			wantStatus: http.StatusOK,
			wantRows:   1,
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(fmt.Sprintf("%s?%s=%s", ledgerPath, reportTypeParam, tt.reportType))
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["Accept"] = tt.format
			res := req.Get()

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}

			if res.Code != http.StatusOK {
				return
			}

			if tt.format == "text/csv" {
				rows := len(strings.Split(res.Body.String(), "\n")) - 1 // don't count empty row at end
				as.Equal(tt.wantRows, rows, "incorrect count of CSV rows")
			} else {
				var ledgerEntries api.LedgerEntries
				as.NoError(json.Unmarshal([]byte(body), &ledgerEntries))
				as.Equal(tt.wantRows, len(ledgerEntries))
			}
		})
	}
}

func (as *ActionSuite) Test_LedgerReconcile() {
	f := as.createFixturesForLedger()
	normalUser := f.Users[0]
	stewardUser := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]

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
			req := as.JSON(ledgerPath)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			res := req.Post(api.LedgerReconcileInput{EndDate: tt.date})

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			for _, s := range tt.wantInBody {
				as.Contains(body, s)
			}

			if res.Code != http.StatusOK {
				return
			}

			var response api.BatchApproveResponse
			err := json.Unmarshal([]byte(body), &response)
			as.NoError(err)

			as.Equal(tt.want, response.NumberOfRecordsApproved, "incorrect number of approved records")

			if tt.want == 0 {
				return
			}

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

	f.Items[0].PaidThroughYear = year
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
			req := as.JSON(ledgerPath + "/annual")
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
	as.NoError(f.Items[2].CreateLedgerEntry(as.DB, models.LedgerEntryTypeCoverageRenewal, 1000))

	return f
}
