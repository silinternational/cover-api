package grifts

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/mail"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
	"github.com/markbates/grift/grift"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

/*
	This is a temporary command-line utility to read a JSON file with data from the legacy Riskman software.

	The input file is expected to have a number of top-level objects, as defined in `LegacyData`. The `Policies`
	list is a complex structure contained related data. The remainder are simple objects.
*/

const (
	MySQLTimeFormat        = "2006-01-02 15:04:05"
	EmptyTime              = "1970-01-01 00:00:00"
	SilenceBadEmailWarning = true
	defaultID              = "9999999999"
	uuidNamespaceConst     = "89cbb2e8-5832-11ec-af6a-95df0dd7b2c5"
)

var trim = strings.TrimSpace

var uuidNamespace uuid.UUID

// userEmailStaffIDMap is a map of email address to staff ID
var userEmailStaffIDMap = map[string]string{}

// userEmailMap is a map of email address to new ID
var userEmailMap = map[string]uuid.UUID{}

// userStaffIDMap is a map of staff ID to new ID
var userStaffIDMap = map[string]uuid.UUID{}

// userStaffIDNames is a map of staff ID to name
var userStaffIDNames = map[string]models.Name{}

// itemIDMap is a map of legacy ID to new ID
var itemIDMap = map[int]uuid.UUID{}

// itemCategoryIDMap is a map of legacy ID to new ID
var itemCategoryIDMap = map[int]uuid.UUID{}

// itemCategoryIDMap is a map of legacy ID to new ID
var riskCategoryMap = map[int]uuid.UUID{}

// householdPolicyMap is a map of household ID to new policy UUID
var householdPolicyMap = map[string]uuid.UUID{}

// entityCodesMap is a map of entity code to entity code UUID
var entityCodesMap = map[string]uuid.UUID{}

// entityCodeNames is a map of entity codes
var entityCodes = map[string]struct{ name, inactive string }{}

// policyUsersCreated is a list of existing PolicyUser records to prevent duplicates
var policyUsersCreated = map[string]struct{}{}

// policyIDMap is a map of legacy ID to new ID
var policyIDMap = map[int]uuid.UUID{}

// time used in place of missing time values
var emptyTime time.Time

var now = time.Now().UTC()

var nPolicyUsersWithStaffID int

var incomeAccounts = map[string]string{
	"HH":  "40200",
	"SIL": "43250",
	"":    "44250",
}

var idpFilenames = map[string]string{
	"SIL":      "./sil-users.csv",
	"USA":      "./usa_idp_users.csv",
	"WPA":      "./wpa_idp_users.csv",
	"Partners": "./partners-users.csv",
}

var workdayFilenames = map[string]string{
	"WPA": "./wpa_workday.csv",
	"SIL": "./sil_workday.csv",
	"USA": "./usa_workday.csv",
}

var _ = grift.Namespace("db", func() {
	_ = grift.Desc("import", "Import legacy data")
	_ = grift.Add("import", func(c *grift.Context) error {
		importStaffIDs()
		readCSVFile("./entity_codes.csv", []int{0, 1, 2}, addEntityCode)

		var obj LegacyData

		f, err := os.Open("./riskman.json")
		if err != nil {
			log.Fatal(err)
		}

		/*  #nosec G307 */
		defer func() {
			if err := f.Close(); err != nil {
				panic("failed to close file, " + err.Error())
			}
		}()

		r := bufio.NewReader(f)
		dec := json.NewDecoder(r)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&obj); err != nil {
			return errors.New("json decode error: " + err.Error())
		}

		fmt.Println("\nJSON record counts: ")
		fmt.Printf("  Admin Users: %d\n", len(obj.Users))
		fmt.Printf("  Policies: %d\n", len(obj.Policies))
		fmt.Printf("  PolicyTypes: %d\n", len(obj.PolicyTypes))
		fmt.Printf("  Maintenance: %d\n", len(obj.Maintenance))
		fmt.Printf("  JournalEntries: %d\n", len(obj.JournalEntries))
		fmt.Printf("  ItemCategories: %d\n", len(obj.ItemCategories))
		fmt.Printf("  RiskCategories: %d\n", len(obj.RiskCategories))
		fmt.Printf("  LossReasons: %d\n", len(obj.LossReasons))
		fmt.Println("")

		if err := models.DB.Transaction(func(tx *pop.Connection) error {
			assignRiskCategoryCostCenters(tx)
			importAdminUsers(tx, obj.Users)
			importItemCategories(tx, obj.ItemCategories)
			importPolicies(tx, obj.Policies)
			importJournalEntries(tx, obj.JournalEntries)

			return nil // errors.New("blocking transaction commit until everything is ready")
		}); err != nil {
			log.Fatalf("failed to import, %s", err)
		}

		return nil
	})
})

func init() {
	emptyTime, _ = time.Parse(MySQLTimeFormat, EmptyTime)
	pop.Debug = false // Disable the Pop log messages
	uuidNamespace = uuid.FromStringOrNil(uuidNamespaceConst)
}

func importStaffIDs() {
	const IDPStaffIDColumn = 0
	const IDPEmailColumn = 1
	const IDPPersonalEmailColumn = 2
	const WorkdayStaffIDColumn = 0
	const WorkdayFirstNameColumn = 1
	const WorkdayLastNameColumn = 2
	const WorkdayEmailColumn = 5
	const WorkdayPersonalEmailColumn = 6

	fmt.Println("\nImporting Workday users")
	for idp, filename := range workdayFilenames {
		n := readCSVFile(filename, []int{
			WorkdayStaffIDColumn, WorkdayEmailColumn,
			WorkdayFirstNameColumn, WorkdayLastNameColumn,
		}, addStaffNameAndID)
		fmt.Printf("  %s Workday users: %d\n", idp, n)
	}

	fmt.Println("\nImporting IDP users")
	for idp, filename := range idpFilenames {
		n := readCSVFile(filename, []int{IDPStaffIDColumn, IDPEmailColumn}, addStaffID)
		fmt.Printf("  %s IDP users: %d\n", idp, n)
	}

	fmt.Println("\nImporting Workday users - personal email addresses")
	for idp, filename := range workdayFilenames {
		n := readCSVFile(filename, []int{WorkdayStaffIDColumn, WorkdayPersonalEmailColumn}, addStaffID)
		fmt.Printf("  %s Workday users: %d\n", idp, n)
	}

	fmt.Println("\nImporting IDP users - personal email addresses")
	for idp, filename := range idpFilenames {
		n := readCSVFile(filename, []int{IDPStaffIDColumn, IDPPersonalEmailColumn}, addStaffID)
		fmt.Printf("  %s IDP users: %d\n", idp, n)
	}

	fmt.Println("\nImporting other user table")
	n := readCSVFile("./other_users.csv", []int{0, 1}, addStaffID)
	fmt.Printf("  other users: %d\n", n)
}

func readCSVFile(filename string, columns []int, storeFunc func([]string) int) int {
	f, err := os.Open(filename) // #nosec G304
	if err != nil {
		log.Fatal(err)
	}

	/*  #nosec G307 */

	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			panic("failed to close file, " + err.Error())
		}
	}(f)

	r := csv.NewReader(bufio.NewReader(f))
	if _, err := r.Read(); err == io.EOF {
		log.Fatalf("empty file '%s'", filename)
	}

	n := 0
	for {
		csvLine, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to read from IDP data file %s, %s", filename, err)
		}

		fields := make([]string, len(columns))
		for i, col := range columns {
			fields[i] = csvLine[col]
		}
		n += storeFunc(fields)
	}
	return n
}

func addStaffID(fields []string) int {
	staffID := fields[0]
	email := fields[1]
	if staffID == "NULL" || staffID == "" || email == "NULL" || email == "" {
		return 0
	}

	trim(email)
	strings.ToLower(email)

	if userEmailStaffIDMap[email] == "" {
		userEmailStaffIDMap[email] = staffID
		return 1
	}

	return 0
}

func addStaffNameAndID(fields []string) int {
	staffID := fields[0]
	firstName := fields[2]
	lastName := fields[3]

	if staffID == "NULL" || staffID == "" {
		return 0
	}

	if _, ok := userStaffIDNames[staffID]; !ok {
		userStaffIDNames[staffID] = models.Name{
			First: firstName,
			Last:  lastName,
		}
	}

	return addStaffID([]string{fields[0], fields[1]})
}

func addEntityCode(fields []string) int {
	code := fields[0]
	name := fields[1]
	inactive := fields[2]
	re, _ := regexp.Compile(`(.*) Company( \(inactive\))?`)
	name = re.ReplaceAllString(name, "$1")

	entityCodes[code] = struct{ name, inactive string }{name, inactive}
	return 1
}

func assignRiskCategoryCostCenters(tx *pop.Connection) {
	mobile := models.RiskCategory{
		ID:         models.RiskCategoryMobileID(),
		CostCenter: "MCMC12",
	}
	if err := tx.UpdateColumns(&mobile, "cost_center"); err != nil {
		log.Fatalf("failed to set cost_center on risk category, %s", err)
	}

	stationary := models.RiskCategory{
		ID:         models.RiskCategoryStationaryID(),
		CostCenter: "MPRO12",
	}
	if err := tx.UpdateColumns(&stationary, "cost_center"); err != nil {
		log.Fatalf("failed to set cost_center on risk category, %s", err)
	}
}

func importAdminUsers(tx *pop.Connection, users []LegacyUser) {
	for _, user := range users {
		user.StaffId = trim(user.StaffId)

		appRole := models.AppRoleCustomer
		if user.Id == "1" {
			appRole = models.AppRoleSignator
		}
		if user.Id == "2" {
			appRole = models.AppRoleSteward
		}

		newUser := models.User{
			ID:            newUUID(user.Email),
			Email:         trim(user.Email),
			EmailOverride: trim(user.EmailOverride),
			FirstName:     trim(user.FirstName),
			LastName:      trim(user.LastName),
			LastLoginUTC:  time.Time(user.LastLoginUtc),
			City:          "Dallas",
			State:         "TX",
			Country:       "USA",
			StaffID:       nulls.NewString(user.StaffId),
			AppRole:       appRole,
			CreatedAt:     time.Time(user.CreatedAt),
		}

		if err := newUser.Create(tx); err != nil {
			log.Fatalf("failed to create user, %s\n%+v", err, newUser)
		}

		if err := newUser.CreateInitialPolicy(tx, ""); err != nil {
			log.Fatalf("failed to create a policy for admin user: %s, %s", newUser.Name(), err)
		}

		userStaffIDMap[user.StaffId] = newUser.ID

		if err := tx.RawQuery("update users set updated_at = ? where id = ?",
			user.UpdatedAt, newUser.ID).Exec(); err != nil {
			log.Fatalf("failed to set updated_at on users, %s", err)
		}
	}
}

func importItemCategories(tx *pop.Connection, in []LegacyItemCategory) {
	for _, category := range in {
		categoryID := stringToInt(category.Id, "ItemCategory ID")

		riskCategoryUUID := getRiskCategoryUUID(category.RiskCategoryId)
		newItemCategory := models.ItemCategory{
			ID:             newUUID(strconv.Itoa(categoryID)),
			RiskCategoryID: riskCategoryUUID,
			Name:           trim(category.Name),
			HelpText:       trim(category.HelpText),
			Status:         getItemCategoryStatus(category),
			AutoApproveMax: fixedPointStringToInt(category.AutoApproveMax, "ItemCategory.AutoApproveMax"),
			LegacyID:       nulls.NewInt(categoryID),
			CreatedAt:      time.Time(category.CreatedAt),
		}
		if category.RiskCategoryId == 2 {
			newItemCategory.RequireMakeModel = true
		}

		if err := newItemCategory.Create(tx); err != nil {
			log.Fatalf("failed to create item category, %s\n%+v", err, newItemCategory)
		}

		itemCategoryIDMap[categoryID] = newItemCategory.ID
		riskCategoryMap[categoryID] = riskCategoryUUID

		if err := tx.RawQuery("update item_categories set updated_at = ? where id = ?",
			category.UpdatedAt, newItemCategory.ID).Exec(); err != nil {
			log.Fatalf("failed to set updated_at on item_categories, %s", err)
		}
	}
}

func getRiskCategoryUUID(legacyID int) uuid.UUID {
	switch legacyID {
	case 1:
		return models.RiskCategoryStationaryID()
	case 2:
		return models.RiskCategoryMobileID()
	}
	log.Printf("unrecognized risk category ID %d", legacyID)
	return models.RiskCategoryMobileID()
}

func getItemCategoryStatus(itemCategory LegacyItemCategory) api.ItemCategoryStatus {
	var status api.ItemCategoryStatus

	switch itemCategory.Status {
	case "enabled":
		status = api.ItemCategoryStatusEnabled

	case "deprecated":
		status = api.ItemCategoryStatusDeprecated

	default:
		log.Printf("unrecognized item category status %s\n", itemCategory.Status)
		status = api.ItemCategoryStatus(itemCategory.Status)
	}

	return status
}

func importPolicies(tx *pop.Connection, policies []LegacyPolicy) {
	var nPolicies, nClaims, nItems, nClaimItems, nDuplicatePolicies, nNamesMatch int
	var nUsers importPolicyUsersResult
	householdsWithMultiplePolicies := map[string]struct{}{}

	for i := range policies {
		normalizePolicy(&policies[i])
		p := policies[i]
		p.HouseholdId = trim(p.HouseholdId)
		p.Notes = trim(p.Notes)

		var policyUUID uuid.UUID

		entityCodeID := getEntityCodeID(tx, p.EntityCode)
		if p.Type == "household" {
			entityCodeID = models.HouseholdEntityID()
		}
		policyID := stringToInt(p.Id, "Policy ID")

		if id, ok := householdPolicyMap[p.HouseholdId]; ok && p.Type == "household" {
			policyUUID = id
			policyIDMap[policyID] = policyUUID
			nDuplicatePolicies++
			householdsWithMultiplePolicies[p.HouseholdId] = struct{}{}
			appendToPolicy(tx, policyUUID, p, policyID)
		} else {
			householdID := nulls.String{}
			if p.HouseholdId != "" {
				householdID = nulls.NewString(p.HouseholdId)
			}

			newPolicy := models.Policy{
				ID:            newUUID(strconv.Itoa(policyID)),
				Name:          trim(p.IdentCode),
				Type:          getPolicyType(p),
				HouseholdID:   householdID,
				CostCenter:    trim(p.CostCenter),
				AccountDetail: trim(p.AccountDetail),
				Account:       strconv.Itoa(p.Account),
				EntityCodeID:  entityCodeID,
				Notes:         p.Notes,
				Email:         p.Email,
				LegacyID:      nulls.NewInt(policyID),
				CreatedAt:     time.Time(p.CreatedAt),
			}
			if newPolicy.Type == api.PolicyTypeHousehold {
				newPolicy.Account = ""
				newPolicy.Name = trim(p.LastName) + " household"
			}
			if err := newPolicy.Create(tx); err != nil {
				log.Fatalf("failed to create policy, %s\n%+v", err, newPolicy)
			}
			policyUUID = newPolicy.ID
			householdPolicyMap[p.HouseholdId] = policyUUID

			if err := tx.RawQuery("update policies set updated_at = ? where id = ?",
				p.UpdatedAt, newPolicy.ID).Exec(); err != nil {
				log.Fatalf("failed to set updated_at on policies, %s", err)
			}

			policyIDMap[policyID] = policyUUID

			nPolicies++
		}

		r := importPolicyUsers(tx, p, policyUUID)
		nUsers.policyUsersCreated += r.policyUsersCreated
		nUsers.usersCreated += r.usersCreated

		nNamesMatch += importItems(tx, policyUUID, policyID, p.Items, r.firstNames)
		nItems += len(p.Items)

		nClaimItems += importClaims(tx, policyUUID, p.Claims)
		nClaims += len(p.Claims)
	}

	fmt.Println("imported: ")
	fmt.Printf("  Policies: %d, Duplicate Household Policies: %d, Households with multiple Policies: %d\n",
		nPolicies, nDuplicatePolicies, len(householdsWithMultiplePolicies))
	fmt.Printf("  Claims: %d\n", nClaims)
	fmt.Printf("  Items: %d with User name: %d\n", nItems, nNamesMatch)
	fmt.Printf("  ClaimItems: %d\n", nClaimItems)
	fmt.Printf("  PolicyUsers: %d w/staffID: %d\n", nUsers.policyUsersCreated, nPolicyUsersWithStaffID)
	fmt.Printf("  Users: %d\n", nUsers.usersCreated)
	fmt.Printf("  Entity Codes: %d\n", len(entityCodesMap))
}

func getEntityCodeID(tx *pop.Connection, code nulls.String) uuid.UUID {
	if !code.Valid {
		return uuid.Nil
	}
	if foundID, ok := entityCodesMap[code.String]; ok {
		return foundID
	}
	entityCodeUUID := importEntityCode(tx, code.String)
	entityCodesMap[code.String] = entityCodeUUID
	return entityCodeUUID
}

func importEntityCode(tx *pop.Connection, code string) uuid.UUID {
	code = trim(code)

	// the source data contains "Yes" for inactive but empty for active
	active := false
	name := code
	if e, ok := entityCodes[code]; ok {
		name = e.name
		if e.inactive != "Yes" {
			active = true
		}
	}

	newEntityCode := models.EntityCode{
		ID:            newUUID(code),
		Code:          code,
		Name:          name,
		Active:        active,
		IncomeAccount: incomeAccount(code),
	}

	if err := newEntityCode.Create(tx); err != nil {
		log.Fatalf("failed to create entity code, %s\n%+v", err, newEntityCode)
	}
	return newEntityCode.ID
}

func appendToPolicy(tx *pop.Connection, policyUUID uuid.UUID, p LegacyPolicy, legacyID int) {
	var policy models.Policy
	if err := policy.FindByID(tx, policyUUID); err != nil {
		log.Fatalf("failed to read existing policy %s", policyUUID)
	}

	newNotes := fmt.Sprintf("%s (ID=%d)", p.Notes, legacyID)
	if policy.Notes != "" {
		policy.Notes += " ---- " + newNotes
	} else {
		policy.Notes = newNotes
	}

	if policy.Email != "" {
		policy.Email += "," + p.Email
	} else {
		policy.Email = p.Email
	}

	if err := tx.UpdateColumns(&policy, "notes", "email"); err != nil {
		panic(err.Error())
	}
}

// getPolicyType gets the correct policy type
func getPolicyType(p LegacyPolicy) api.PolicyType {
	var policyType api.PolicyType

	switch p.Type {
	case "household":
		policyType = api.PolicyTypeHousehold
	case "ou", "corporate":
		policyType = api.PolicyTypeTeam
	default:
		log.Fatalf("no policy type in policy '" + p.Id + "'")
	}

	return policyType
}

// normalizePolicy adjusts policy fields to pass validation checks
func normalizePolicy(p *LegacyPolicy) {
	if p.Type == "household" {
		p.CostCenter = ""
		p.EntityCode = nulls.String{}
	}

	if p.Type == "ou" || p.Type == "corporate" {
		p.HouseholdId = ""

		if !p.EntityCode.Valid || p.EntityCode.String == "" {
			p.EntityCode = nulls.NewString(defaultID)
			log.Printf("Policy[%s] EntityCode is empty, using %s", p.Id, defaultID)
		}
		if p.CostCenter == "" {
			p.CostCenter = defaultID
			log.Printf("Policy[%s] CostCenter is empty, using %s", p.Id, defaultID)
		}
	}
}

type importPolicyUsersResult struct {
	policyUsersCreated int
	firstNames         map[uuid.UUID]string
	usersCreated       int
}

func importPolicyUsers(tx *pop.Connection, p LegacyPolicy, policyID uuid.UUID) importPolicyUsersResult {
	result := importPolicyUsersResult{firstNames: map[uuid.UUID]string{}}

	if p.Email == "" {
		if !SilenceBadEmailWarning {
			log.Printf("Policy[%s] Email is empty\n", p.Id)
		}
		return result
	}

	s := strings.Split(strings.ReplaceAll(p.Email, " ", ","), ",")
	for _, email := range s {
		validEmail, ok := validMailAddress(email)
		if !ok {
			if !SilenceBadEmailWarning {
				log.Printf("Policy[%s] has an invalid email address: '%s'\n", p.Id, email)
			}
			continue
		}
		r := createPolicyUser(tx, validEmail, p.FirstName, p.LastName, policyID)
		result.policyUsersCreated += r.policyUsersCreated
		result.usersCreated += r.usersCreated
		result.firstNames[r.userID] = p.FirstName
	}
	return result
}

type createPolicyUserResult struct {
	policyUsersCreated int
	usersCreated       int
	userID             uuid.UUID
}

func createPolicyUser(tx *pop.Connection, email, firstName, lastName string, policyID uuid.UUID) createPolicyUserResult {
	result := createPolicyUserResult{}

	email = strings.ToLower(email)
	userID, ok := userEmailMap[email]
	if !ok {
		result.usersCreated, userID = createUserFromEmailAddress(tx, email, firstName, lastName)
	}

	if _, ok = policyUsersCreated[policyID.String()+userID.String()]; ok {
		result.userID = userID
		return result
	}

	policyUser := models.PolicyUser{
		ID:       newUUID(policyID.String() + userID.String()),
		PolicyID: policyID,
		UserID:   userID,
	}
	if err := policyUser.Create(tx); err != nil {
		log.Fatalf("failed to create new PolicyUser, %s", err)
	}
	result.policyUsersCreated = 1

	policyUsersCreated[policyID.String()+userID.String()] = struct{}{}

	result.userID = userID
	return result
}

func createUserFromEmailAddress(tx *pop.Connection, email, firstName, lastName string) (int, uuid.UUID) {
	var staffID nulls.String
	if id, ok := userEmailStaffIDMap[email]; ok {
		staffID = nulls.NewString(id)

		nPolicyUsersWithStaffID++

		// use name imported from Workday
		name := userStaffIDNames[id]
		if name.First != "" {
			firstName = name.First
		}
		if name.Last != "" {
			lastName = name.Last
		}
	}

	// check for existing user with this staff ID
	if staffID.Valid {
		if userID, ok := userStaffIDMap[staffID.String]; ok {
			return 0, userID
		}
	}
	user := models.User{
		ID:           newUUID(email),
		Email:        email,
		FirstName:    trim(firstName),
		LastName:     trim(lastName),
		StaffID:      staffID,
		AppRole:      models.AppRoleCustomer,
		LastLoginUTC: emptyTime,
	}
	if err := user.Create(tx); err != nil {
		log.Fatalf("failed to create new User for policy, %s", err)
	}
	userStaffIDMap[staffID.String] = user.ID
	userEmailMap[email] = user.ID
	return 1, user.ID
}

func validMailAddress(address string) (string, bool) {
	if strings.HasSuffix(address, "@sil") {
		address = address + ".org"
	}
	addr, err := mail.ParseAddress(address)
	if err != nil {
		return "", false
	}
	return addr.Address, true
}

func importClaims(tx *pop.Connection, policyID uuid.UUID, claims []LegacyClaim) int {
	nClaimItems := 0

	for _, c := range claims {
		claimID := stringToInt(c.Id, "Claim ID")
		claimDesc := fmt.Sprintf("Claim[%d].", claimID)

		newClaim := models.Claim{
			ID:                  newUUID(strconv.Itoa(claimID)),
			LegacyID:            nulls.NewInt(claimID),
			PolicyID:            policyID,
			IncidentDate:        time.Time(c.IncidentDate),
			IncidentType:        getIncidentType(c, claimID),
			IncidentDescription: getIncidentDescription(c),
			Status:              api.ClaimStatusPaid,
			ReviewDate:          nulls.Time(c.ReviewDate),
			ReviewerID:          getAdminUserUUID(strconv.Itoa(c.ReviewerId), claimDesc+"ReviewerID"),
			PaymentDate:         nulls.Time(c.PaymentDate),
			TotalPayout:         fixedPointStringToCurrency(c.TotalPayout, "Claim.TotalPayout"),
			City:                trim(c.City),
			CreatedAt:           time.Time(c.CreatedAt),
		}

		newClaim.State, newClaim.Country = getStateAndCountry(c.Country)

		if err := newClaim.Create(tx); err != nil {
			log.Fatalf("failed to create claim, %s\n%+v", err, newClaim)
		}

		if err := tx.RawQuery("update claims set updated_at = ? where id = ?",
			c.UpdatedAt, newClaim.ID).Exec(); err != nil {
			log.Fatalf("failed to set updated_at on claims, %s", err)
		}

		importClaimItems(tx, newClaim, c.ClaimItems)
		nClaimItems += len(c.ClaimItems)
	}

	return nClaimItems
}

func importClaimItems(tx *pop.Connection, claim models.Claim, items []LegacyClaimItem) {
	for _, c := range items {
		claimItemID := stringToInt(c.Id, "ClaimItem ID")
		itemDesc := fmt.Sprintf("Claim[%d] ClaimItem[%d] ", claim.LegacyID.Int, claimItemID)

		itemUUID, ok := itemIDMap[c.ItemId]
		if !ok {
			log.Fatalf("item ID %d not found in claim %d item list", claimItemID, claim.LegacyID.Int)
		}

		newClaimItem := models.ClaimItem{
			ID:              newUUID(strconv.Itoa(claimItemID)),
			ClaimID:         claim.ID,
			ItemID:          itemUUID,
			IsRepairable:    getIsRepairable(c),
			RepairEstimate:  fixedPointStringToCurrency(c.RepairEstimate, "ClaimItem.RepairEstimate"),
			RepairActual:    fixedPointStringToCurrency(c.RepairActual, "ClaimItem.RepairActual"),
			ReplaceEstimate: fixedPointStringToCurrency(c.ReplaceEstimate, "ClaimItem.ReplaceEstimate"),
			ReplaceActual:   fixedPointStringToCurrency(c.ReplaceActual, "ClaimItem.ReplaceActual"),
			PayoutOption:    getPayoutOption(c.PayoutOption, itemDesc+"PayoutOption"),
			PayoutAmount:    fixedPointStringToCurrency(c.PayoutAmount, "ClaimItem.PayoutAmount"),
			FMV:             fixedPointStringToCurrency(c.Fmv, "ClaimItem.FMV"),
			LegacyID:        nulls.NewInt(claimItemID),
			CreatedAt:       time.Time(c.CreatedAt),
			City:            trim(c.City),
		}

		newClaimItem.State, newClaimItem.Country = getStateAndCountry(c.Country)

		if err := newClaimItem.Create(tx); err != nil {
			log.Fatalf("failed to create claim item %d, %s\nClaimItem:\n%+v", claimItemID, err, newClaimItem)
		}

		// use CreatedAt because the source data has no UpdatedAt
		if err := tx.RawQuery("update claim_items set updated_at = ? where id = ?",
			c.CreatedAt, newClaimItem.ID).Exec(); err != nil {
			log.Fatalf("failed to set updated_at on claim_items, %s", err)
		}

	}
}

func getPayoutOption(s, desc string) api.PayoutOption {
	var option api.PayoutOption

	if s == "" {
		log.Printf("%s is empty, setting to 'FMV'", desc)
		return api.PayoutOptionFMV
	}

	switch s {
	case "repair":
		option = api.PayoutOptionRepair
	case "replace":
		option = api.PayoutOptionReplacement
	case "fmv":
		option = api.PayoutOptionFMV
	default:
		log.Fatalf("%s unrecognized payout option: '%s'\n", desc, s)
	}

	return option
}

func getIsRepairable(c LegacyClaimItem) bool {
	if c.IsRepairable != 0 && c.IsRepairable != 1 {
		log.Println("ClaimItem.IsRepairable is neither 0 nor 1")
	}
	return c.IsRepairable == 1
}

func getAdminUserUUID(staffID, desc string) nulls.UUID {
	userUUID, ok := userStaffIDMap[staffID]
	if !ok {
		log.Printf("%s has unrecognized staff ID %s\n", desc, staffID)
		return nulls.UUID{}
	}
	return nulls.NewUUID(userUUID)
}

func getIncidentType(claim LegacyClaim, claimID int) api.ClaimIncidentType {
	var incidentType api.ClaimIncidentType

	switch claim.IncidentType {
	case "Broken", "Dropped":
		incidentType = api.ClaimIncidentTypePhysicalDamage
	case "Lightning", "Lightening":
		incidentType = api.ClaimIncidentTypeElectricalSurge
	case "Theft":
		incidentType = api.ClaimIncidentTypeTheft
	case "Water Damage":
		incidentType = api.ClaimIncidentTypeWaterDamage
	case "Fire":
		incidentType = api.ClaimIncidentTypeFireDamage
	case "War":
		incidentType = api.ClaimIncidentTypeEvacuation
	case "Miscellaneous", "Unknown", "Vandalism":
		incidentType = api.ClaimIncidentTypeOther
	default:
		log.Printf("Claim[%d] has unrecognized incident type ('%s'), using \"Other\"\n", claimID, claim.IncidentType)
		incidentType = api.ClaimIncidentTypeOther
	}

	return incidentType
}

func getIncidentDescription(claim LegacyClaim) string {
	if claim.IncidentDescription == "" {
		return "[no description provided]"
	}
	return claim.IncidentDescription
}

// importItems loads legacy items onto a policy. Returns true if at least one item is approved.
func importItems(tx *pop.Connection, policyUUID uuid.UUID, policyID int, items []LegacyItem,
	names map[uuid.UUID]string) int {
	nNamesMatch := 0
	for _, item := range items {
		itemID := stringToInt(item.Id, "Item ID")
		itemDesc := fmt.Sprintf("Policy[%d] Item[%d] ", policyID, itemID)

		newItem := models.Item{
			ID:                newUUID(strconv.Itoa(itemID)),
			Name:              trim(item.Name),
			CategoryID:        itemCategoryIDMap[item.CategoryId],
			RiskCategoryID:    riskCategoryMap[item.CategoryId],
			PolicyID:          policyUUID,
			Make:              trim(item.Make),
			Model:             trim(item.Model),
			SerialNumber:      trim(item.SerialNumber),
			CoverageAmount:    fixedPointStringToInt(item.CoverageAmount, itemDesc+"CoverageAmount"),
			CoverageStatus:    getCoverageStatus(item),
			CoverageStartDate: time.Time(item.CreatedAt),
			CoverageEndDate:   nulls.Time(item.CoverageEndDate),
			LegacyID:          nulls.NewInt(itemID),
			City:              trim(item.City),
			CreatedAt:         time.Time(item.CreatedAt),
		}

		newItem.State, newItem.Country = getStateAndCountry(item.Country)

		for id, name := range names {
			if name != "" && strings.Contains(strings.ToLower(newItem.Name), strings.ToLower(name)) {
				newItem.PolicyUserID = nulls.NewUUID(id)
				nNamesMatch++
				break
			}
		}
		if item.CoverageEndDate.Valid {
			newItem.PaidThroughYear = item.CoverageEndDate.Time.Year()
		} else if newItem.CoverageAmount > 0 {
			newItem.PaidThroughYear = now.Year()
		}

		if err := newItem.Create(tx); err != nil {
			log.Fatalf("failed to create item, %s\n%+v", err, newItem)
		}
		itemIDMap[itemID] = newItem.ID

		if err := tx.RawQuery("update items set updated_at = ? where id = ?",
			item.UpdatedAt, newItem.ID).Exec(); err != nil {
			log.Fatalf("failed to set updated_at on item, %s", err)
		}
	}
	return nNamesMatch
}

func getCoverageStatus(item LegacyItem) api.ItemCoverageStatus {
	var coverageStatus api.ItemCoverageStatus

	switch item.CoverageStatus {
	case "approved":
		coverageStatus = api.ItemCoverageStatusApproved

	case "inactive":
		coverageStatus = api.ItemCoverageStatusInactive

	default:
		log.Fatalf("unknown coverage status %s\n", item.CoverageStatus)
	}

	return coverageStatus
}

func importJournalEntries(tx *pop.Connection, entries []JournalEntry) {
	// fmt.Printf(`"%s","%s","%s","%s","%s","%s","%s","%s","%s","%s","%s","%s","%s","%s"`+"\n",
	//	"PolicyID", "PolicyType", "FirstName", "LastName", "JERecType", "AccCostCtr1", "AccCostCtr2", "CustJE",
	//	"DateEntd", "DateSubm", "Entity", "AccNum", "RMJE, ", "JERecNum")

	nImported := 0
	badPolicyIDs := map[int]struct{}{}
	for _, e := range entries {
		//	fmt.Printf(`%d,%d,"%s","%s",%d,"%s","%s",%f,"%s","%s","%s",%d,%f,"%s"`+"\n",
		//		e.PolicyID, e.PolicyType, e.FirstName, e.LastName, e.JERecType, e.AccCostCtr1, e.AccCostCtr2, e.CustJE,
		//		e.DateEntd, e.DateSubm, e.Entity, e.AccNum, e.RMJE, e.JERecNum)
		//}

		policyType := api.PolicyTypeTeam
		if e.Entity == "MMB/STM" {
			policyType = api.PolicyTypeHousehold
		}

		policyUUID, err := getPolicyUUID(e.PolicyID)
		if err != nil {
			badPolicyIDs[e.PolicyID] = struct{}{}
			continue
		}
		submitted := emptyTime
		if e.DateSubm.Valid {
			submitted = e.DateSubm.Time
		}
		if e.Entity == "MMB/STM" {
			e.Entity = "HH"
		}
		l := models.LedgerEntry{
			ID:               newUUID(e.JERecNum),
			PolicyID:         policyUUID,
			Amount:           api.Currency(math.Round(e.CustJE * domain.CurrencyFactor)),
			DateSubmitted:    submitted,
			DateEntered:      nulls.Time(e.DateEntd),
			LegacyID:         nulls.NewInt(stringToInt(e.JERecNum, "LedgerEntry.LegacyID")),
			Type:             getLedgerEntryType(e.JERecType),
			RiskCategoryName: policyTypeToRiskCategoryName(e.PolicyType),
			RiskCategoryCC:   policyTypeToRiskCategoryCC(e.PolicyType),
			PolicyType:       policyType,
			AccountNumber:    strconv.Itoa(e.AccNum),
			HouseholdID:      trim(e.AccCostCtr1),
			CostCenter:       trim(e.AccCostCtr2),
			EntityCode:       trim(e.Entity),
			IncomeAccount:    incomeAccount(e.Entity),
			FirstName:        trim(e.FirstName),
			LastName:         trim(e.LastName),
		}
		l.CreatedAt = l.DateSubmitted

		if err := l.Create(tx); err != nil {
			log.Fatalf("failed to create ledger entry, %s\n%+v", err, l)
		}

		updated := l.DateSubmitted
		if l.DateEntered.Valid {
			updated = l.DateEntered.Time
		}
		if err = tx.RawQuery("update ledger_entries set updated_at = ? where id = ?",
			updated, l.ID).Exec(); err != nil {
			log.Fatalf("failed to set updated_at on ledger_entries, %s", err)
		}

		nImported++
	}

	s := make([]string, len(badPolicyIDs))
	i := 0
	for id := range badPolicyIDs {
		s[i] = fmt.Sprintf("%v", id)
		i++
	}
	fmt.Printf("  LedgerEntries: %d (policy IDs not found: %s)\n", nImported, strings.Join(s, ","))
}

func getLedgerEntryType(i int) models.LedgerEntryType {
	types := map[int]models.LedgerEntryType{
		1:  models.LedgerEntryTypeNewCoverage,
		2:  models.LedgerEntryTypeCoverageChange,
		3:  models.LedgerEntryTypePolicyAdjustment,
		4:  models.LedgerEntryTypeClaim,
		5:  models.LedgerEntryTypeLegacy5,
		6:  models.LedgerEntryTypeClaimAdjustment,
		20: models.LedgerEntryTypeLegacy20,
	}
	return types[i]
}

func policyTypeToRiskCategoryName(i int) string {
	names := map[int]string{
		1: "Mobile",
		2: "Stationary",
	}
	return names[i]
}

func policyTypeToRiskCategoryCC(i int) string {
	costCenter := map[int]string{
		1: "MCMC12",
		2: "MPRO12",
	}
	return costCenter[i]
}

func getPolicyUUID(id int) (uuid.UUID, error) {
	if u, ok := policyIDMap[id]; ok {
		return u, nil
	}
	return uuid.UUID{}, fmt.Errorf("bad policy ID %d", id)
}

func stringToInt(s, msg string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("%s '%s' is not an int", msg, s)
	}
	return n
}

func fixedPointStringToCurrency(s, desc string) api.Currency {
	return api.Currency(fixedPointStringToInt(s, desc))
}

func fixedPointStringToInt(s, desc string) int {
	if s == "" {
		log.Printf("%s is empty, setting to 0", desc)
		return 0
	}

	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		log.Fatalf("%s has more than one '.' character: '%s'", desc, s)
	}
	intPart := stringToInt(parts[0], desc+" left of decimal")
	if len(parts[1]) != 2 {
		log.Fatalf("%s does not have two digits after the decimal: %s", desc, s)
	}
	fractionalPart := stringToInt(parts[1], desc+" right of decimal")
	return intPart*100 + fractionalPart
}

var states = map[string]string{
	"Alabama":              "AL",
	"Alaska":               "AK",
	"Arizona":              "AZ",
	"Arkansas":             "AR",
	"California":           "CA",
	"Colorado":             "CO",
	"Connecticut":          "CT",
	"Delaware":             "DE",
	"District of Columbia": "DC",
	"Florida":              "FL",
	"Georgia":              "GA",
	"Hawaii":               "HI",
	"Idaho":                "ID",
	"Illinois":             "IL",
	"Indiana":              "IN",
	"Iowa":                 "IA",
	"Kansas":               "KS",
	"Kentucky":             "KY",
	"Louisiana":            "LA",
	"Maine":                "ME",
	"Montana":              "MT",
	"Nebraska":             "NE",
	"Nevada":               "NV",
	"New Hampshire":        "NH",
	"New Jersey":           "NJ",
	"New Mexico":           "NM",
	"New York":             "NY",
	"North Carolina":       "NC",
	"North Dakota":         "ND",
	"Ohio":                 "OH",
	"Oklahoma":             "OK",
	"Oregon":               "OR",
	"Maryland":             "MD",
	"Massachusetts":        "MA",
	"Michigan":             "MI",
	"Minnesota":            "MN",
	"Mississippi":          "MS",
	"Missouri":             "MO",
	"Pennsylvania":         "PA",
	"Rhode Island":         "RI",
	"South Carolina":       "SC",
	"South Dakota":         "SD",
	"Tennessee":            "TN",
	"Texas":                "TX",
	"Utah":                 "UT",
	"Vermont":              "VT",
	"Virginia":             "VA",
	"Washington":           "WA",
	"West Virginia":        "WV",
	"Wisconsin":            "WI",
	"Wyoming":              "WY",
	"Alberta":              "AB",
	"Ontario":              "ON",
}

func getStateAndCountry(c string) (state, country string) {
	c = trim(c)

	if abbr, ok := states[c]; ok {
		state = abbr
		switch state {
		case "AB", "ON":
			country = "Canada"
		default:
			country = "United States of America"
		}
		return
	}

	if len(c) == 2 {
		abbr := strings.ToUpper(c)
		switch abbr {
		case "CH":
			country = "Switzerland"
		case "FR":
			country = "France"
		case "NZ":
			country = "New Zealand"
		case "UK":
			country = "United Kingdom"
		case "AB", "BC", "MB", "NB", "NL", "NT", "NS", "NU", "ON", "PE", "QC", "SK", "YT":
			state = abbr
			country = "Canada"
		case "US":
			state = ""
			country = "United States of America"
		default:
			state = abbr
			country = "United States of America"
		}
		return
	}

	state = ""
	country = c
	return
}

func incomeAccount(entityCode string) string {
	account := incomeAccounts[trim(entityCode)]
	if account == "" {
		account = incomeAccounts[""]
	}
	return account
}

func newUUID(seed string) uuid.UUID {
	return uuid.NewV5(uuidNamespace, seed)
}
