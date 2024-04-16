package actions

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/labstack/echo/v4"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

const (
	// http param for month (number)
	MonthParam = "month"

	// http param for year
	YearParam = "year"
)

// swagger:operation GET /policies Policies PoliciesList
// PoliciesList
//
// Get the data for all the user's Policies if the user is not an admin. If called by an admin, returns all Policies
// in the system, limited by query parameters.
// ---
//
//	parameters:
//	- name: limit
//	  in: query
//	  required: false
//	  description: number of records to return, minimum 1, maximum 50, default 10
//	- name: search
//	  in: query
//	  required: false
//	  description: search text to find across fields (name, household_id, cost_center, and all members' first and last names)
//	- name: filter
//	  in: query
//	  required: false
//	  description: comma-separated list of search pairs like "field:text". Presently, only meta-field 'active' is supported
//	responses:
//	  '200':
//	    description: all policies
//	    schema:
//	      type: object
//	      properties:
//	        meta:
//	          "$ref": "#/definitions/Meta"
//	        data:
//	          "$ref": "#/definitions/Policies"
func policiesList(c echo.Context) error {
	user := models.CurrentUser(c)

	if user.IsAdmin() {
		return policiesListAdmin(c)
	}

	return policiesListCustomer(c)
}

func policiesListAdmin(c echo.Context) error {
	tx := models.Tx(c)
	var policies models.Policies

	qp := api.NewQueryParams(c.QueryParams())
	p, err := policies.Query(tx, qp)
	if err != nil {
		return reportError(c, err)
	}

	response := api.ListResponse{
		Data: policies.ConvertToAPI(tx),
		Meta: api.Meta{Paginator: p},
	}

	return renderOk(c, response)
}

func policiesListCustomer(c echo.Context) error {
	tx := models.Tx(c)
	user := models.CurrentUser(c)

	user.LoadPolicies(tx, false)

	response := api.ListResponse{
		Data: user.Policies.ConvertToAPI(tx),
	}

	return renderOk(c, response)
}

// swagger:operation GET /policies/{id} Policies PoliciesView
// PoliciesView
//
// gets the data for a specific policy
// ---
//
//	responses:
//	  '200':
//	    description: a policy
//	    schema:
//	      "$ref": "#/definitions/Policy"
func policiesView(c echo.Context) error {
	policy := getReferencedPolicyFromCtx(c)

	return renderOk(c, policy.ConvertToAPI(models.Tx(c), true))
}

// swagger:operation POST /policies Policies PoliciesCreateTeam
// PoliciesCreateTeam
//
// create a new Policy with type Team
// ---
//
//	parameters:
//	  - name: policy input
//	    in: body
//	    description: policy create input object
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/PolicyCreate"
//	responses:
//	  '200':
//	    description: the new Policy
//	    schema:
//	      "$ref": "#/definitions/Policy"
func policiesCreateTeam(c echo.Context) error {
	var input api.PolicyCreate
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)

	policy := models.Policy{
		Name:          input.Name,
		CostCenter:    input.CostCenter,
		Account:       input.Account,
		AccountDetail: input.AccountDetail,
		EntityCodeID:  models.EntityCodeID(input.EntityCode),
	}

	if err := policy.CreateTeam(c); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, policy.ConvertToAPI(tx, true))
}

// swagger:operation PUT /policies/{id} Policies PoliciesUpdate
// PoliciesUpdate
//
// update a policy
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: policy ID
//	  - name: policy update input
//	    in: body
//	    description: policy update input object
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/PolicyUpdate"
//	responses:
//	  '200':
//	    description: updated Policy
//	    schema:
//	      "$ref": "#/definitions/Policy"
func policiesUpdate(c echo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)

	var update api.PolicyUpdate
	if err := StrictBind(c, &update); err != nil {
		return reportError(c, err)
	}

	switch policy.Type {
	case api.PolicyTypeHousehold:
		if update.HouseholdID != nil {
			policy.HouseholdID = nulls.NewString(*update.HouseholdID)
		}
		policy.CostCenter = ""
		policy.Account = ""
		policy.AccountDetail = ""
		policy.EntityCodeID = models.HouseholdEntityID()
	case api.PolicyTypeTeam:
		policy.HouseholdID = nulls.String{}
		policy.CostCenter = update.CostCenter
		policy.Account = update.Account
		policy.AccountDetail = update.AccountDetail
		policy.EntityCodeID = models.EntityCodeID(update.EntityCode)
	}

	policy.Name = update.Name

	if err := policy.Update(c); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, policy.ConvertToAPI(tx, true))
}

// swagger:operation POST /policies/{id}/ledger-reports PolicyLedgerReport PolicyLedgerReportCreate
// PolicyLedgerReportCreate
//
// Create and return a report on the ledger entries of a policy as specified by the input object.
// The returned object contains metadata and a File object pointing to a CSV file.
// If no ledger entries are found with a `date_entered` value that matches the requested
// Type, Year and (if applicable) Month, then a 204 is returned.
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: policy ID
//	  - name: input
//	    in: body
//	    description: PolicyLedgerReportCreateInput object
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/PolicyLedgerReportCreateInput"
//	responses:
//	  '200':
//	    description: the requested LedgerReport for the Policy
//	    schema:
//	      type: array
//	      items:
//	        "$ref": "#/definitions/LedgerReport"
func policiesLedgerReportCreate(c echo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)

	var input api.PolicyLedgerReportCreateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	report, err := models.NewPolicyLedgerReport(c, *policy, input.Type, input.Month, input.Year)
	if err != nil {
		return reportError(c, err)
	}

	if len(report.LedgerEntries) == 0 {
		return c.JSON(http.StatusNoContent, nil)
	}

	if err = report.Create(tx); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, report.ConvertToAPI(tx))
}

// swagger:operation GET /policies/{id}/ledger-reports PolicyLedgerReport PolicyLedgerTableView
// PolicyLedgerTableView
//
// Return data regarding the ledger entries of a policy for a particular month.
// If no ledger entries are found with a `date_entered` value that matches the requested
// Year and Month, then a 204 is returned.
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: policy ID
//	  - name: month
//	    in: query
//	    required: true
//	    description: the month (number) within which the ledger entries were entered into the accounting system
//	  - name: year
//	    in: query
//	    required: true
//	    description: the year within which the ledger entries were entered into the accounting system
//	responses:
//	  '200':
//	    description: the requested LedgerTable for the Policy
//	    schema:
//	      "$ref": "#/definitions/LedgerTable"
func policiesLedgerTableView(c echo.Context) error {
	policy := getReferencedPolicyFromCtx(c)

	m := c.QueryParam(MonthParam)
	month, err := strconv.Atoi(m)
	if err != nil {
		appErr := api.AppError{
			HttpStatus: http.StatusBadRequest,
			Key:        api.ErrorInvalidDate,
			Message:    fmt.Sprintf("%s, %s, must be a number", MonthParam, m),
		}
		return reportError(c, &appErr)
	}

	y := c.QueryParam(YearParam)
	year, err := strconv.Atoi(y)
	if err != nil {
		appErr := api.AppError{
			HttpStatus: http.StatusBadRequest,
			Key:        api.ErrorInvalidDate,
			Message:    fmt.Sprintf("%s, %s, must be a number", YearParam, y),
		}
		return reportError(c, &appErr)
	}

	table, err := models.PolicyLedgerTable(c, *policy, month, year)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, table)
}

// swagger:operation POST /policies/{id}/strikes PolicyStrike PolicyStrikeCreate
// PolicyStrikeCreate
//
// Create a strike on the policy and return its recent strikes
// ---
//
//	parameters:
//	  - name: id
//	    in: path
//	    required: true
//	    description: policy ID
//	  - name: input
//	    in: body
//	    description: StrikeInput object
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/StrikeInput"
//	responses:
//	  '200':
//	    description: the Strikes for the Policy that are still in force
//	    schema:
//	      type: array
//	      items:
//	        "$ref": "#/definitions/Strike"
func policiesStrikeCreate(c echo.Context) error {
	tx := models.Tx(c)
	policy := getReferencedPolicyFromCtx(c)

	var input api.StrikeInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	strike := models.Strike{
		Description: input.Description,
		PolicyID:    policy.ID,
	}

	if err := strike.Create(tx); err != nil {
		return reportError(c, err)
	}

	// Just to be extra careful add a few minutes
	soon := time.Now().UTC().Add(time.Minute * 3)

	var strikes models.Strikes
	if err := strikes.RecentForPolicy(tx, policy.ID, soon); err != nil {
		reportError(c, err)
	}

	return renderOk(c, strikes.ConvertToAPI(tx))
}

// swagger:operation POST /policies/{id}/imports Policies PoliciesImport
// PoliciesImport
//
// Import policies and items from a CSV file. This endpoint is initially for importing vehicle information
// from a legacy system. It may be removed at a later time, or it may be adapted for other types of data.
// ---
//
//	consumes:
//	  - multipart/form-data
//	parameters:
//	  - name: file
//	    in: formData
//	    type: file
//	    description: file object
//	responses:
//	  '200':
//	    description: uploaded File data
//	    schema:
//	      "$ref": "#/definitions/PoliciesImportResponse"
func policiesImport(c echo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user is not allowed to import policies")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	tx := models.Tx(c)
	f, err := c.FormFile(fileFieldName)
	if err != nil {
		err := fmt.Errorf("error getting uploaded file from context ... %v", err)
		return reportError(c, api.NewAppError(err, api.ErrorReceivingFile, api.CategoryInternal))
	}

	if f.Size > int64(domain.MaxFileSize) {
		err := fmt.Errorf("file upload size (%v) greater than max (%v)", f.Size, domain.MaxFileSize)
		return reportError(c, api.NewAppError(err, api.ErrorStoreFileTooLarge, api.CategoryUser))
	}

	file, err := f.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	response, err := models.ImportPolicies(tx, file)
	if err != nil {
		return reportError(c, err)
	}

	return renderOk(c, response)
}

// getReferencedPolicyFromCtx pulls the models.Policy resource from context that was put there
// by the AuthZ middleware
func getReferencedPolicyFromCtx(c echo.Context) *models.Policy {
	policy, ok := c.Get(domain.TypePolicy).(*models.Policy)
	if !ok {
		panic("policy not found in context")
	}
	return policy
}
