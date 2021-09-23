package grifts

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/mail"
	"os"
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
	TimeFormat               = "2006-01-02 15:04:05"
	EmptyTime                = "1970-01-01 00:00:00"
	SilenceEmptyTimeWarnings = true
	SilenceBadEmailWarning   = true
)

const (
	SILUsersFilename      = "./sil-users.csv"
	PartnersUsersFilename = "./partners-users.csv"
)

var trim = strings.TrimSpace

// userEmailStaffIDMap is a map of email address to staff ID
var userEmailStaffIDMap = map[string]string{}

// userEmailMap is a map of email address to new ID
var userEmailMap = map[string]uuid.UUID{}

// userStaffIDMap is a map of staff ID to new ID
var userStaffIDMap = map[string]uuid.UUID{}

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

// policyUserMap is a list of existing PolicyUser records to prevent duplicates
var policyUserMap = map[string]struct{}{}

// policyNotes is used for accumulating policy notes as policies are merged
var policyNotes = map[uuid.UUID]string{}

// policyIDMap is a map of legacy ID to new ID
var policyIDMap = map[int]uuid.UUID{}

// activeEntities is used to track which entity codes are used on active policies
var activeEntities = map[uuid.UUID]struct{}{}

// time used in place of missing time values
var emptyTime time.Time

var nPolicyUsersWithStaffID int

var _ = grift.Namespace("db", func() {
	_ = grift.Desc("import", "Import legacy data")
	_ = grift.Add("import", func(c *grift.Context) error {
		importIdpUsers()

		var obj LegacyData

		f, err := os.Open("./riskman.json")
		if err != nil {
			log.Fatal(err)
		}
		defer func(f *os.File) {
			if err := f.Close(); err != nil {
				panic("failed to close file, " + err.Error())
			}
		}(f)

		r := bufio.NewReader(f)
		dec := json.NewDecoder(r)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&obj); err != nil {
			return errors.New("json decode error: " + err.Error())
		}

		fmt.Println("record counts: ")
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
	emptyTime, _ = time.Parse(TimeFormat, EmptyTime)
	pop.Debug = false // Disable the Pop log messages
}

func importIdpUsers() {
	nSilUsers := importIdpUsersFromFile("SIL")
	fmt.Printf("SIL users: %d\n", nSilUsers)

	nPartnersUsers := importIdpUsersFromFile("Partners")
	fmt.Printf("Partners users: %d\n", nPartnersUsers)
}

func importIdpUsersFromFile(idpName string) int {
	filenames := map[string]string{"SIL": SILUsersFilename, "Partners": PartnersUsersFilename}
	f, err := os.Open(filenames[idpName])
	if err != nil {
		log.Fatal(err)
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			panic("failed to close file, " + err.Error())
		}
	}(f)

	r := csv.NewReader(bufio.NewReader(f))

	n := 0
	for {
		csvLine, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to read from IDP data file %s, %s", filenames[idpName], err)
		}

		staffID := csvLine[2]
		email := csvLine[7]
		if staffID == "NULL" || email == "NULL" {
			continue
		}

		if userEmailStaffIDMap[strings.ToLower(email)] == "" {
			userEmailStaffIDMap[strings.ToLower(email)] = staffID
			n++
		} else {
			if userEmailStaffIDMap[strings.ToLower(email)] != staffID {
				log.Fatalf("in file %s, email address '%s' maps to two different staff IDs: '%s' and '%s'",
					filenames[idpName], email, userEmailStaffIDMap[strings.ToLower(email)], staffID)
			}
		}
	}
	return n
}

func importAdminUsers(tx *pop.Connection, users []LegacyUser) {
	for _, user := range users {
		userID := stringToInt(user.Id, "User ID")
		userDesc := fmt.Sprintf("User[%d].", userID)

		user.StaffId = trim(user.StaffId)

		appRole := models.AppRoleSteward
		if user.Id == "1" {
			appRole = models.AppRoleSignator
		}

		newUser := models.User{
			Email:         trim(user.Email),
			EmailOverride: trim(user.EmailOverride),
			FirstName:     trim(user.FirstName),
			LastName:      trim(user.LastName),
			LastLoginUTC:  parseStringTime(user.LastLoginUtc, userDesc+"LastLoginUTC"),
			Location:      trim(user.Location),
			StaffID:       user.StaffId,
			AppRole:       appRole,
			CreatedAt:     parseStringTime(user.CreatedAt, userDesc+"CreatedAt"),
		}

		if err := newUser.Create(tx); err != nil {
			log.Fatalf("failed to create user, %s\n%+v", err, newUser)
		}

		userStaffIDMap[user.StaffId] = newUser.ID

		if err := tx.RawQuery("update users set updated_at = ? where id = ?",
			parseStringTime(user.UpdatedAt, userDesc+"UpdatedAt"), newUser.ID).Exec(); err != nil {
			log.Fatalf("failed to set updated_at on users, %s", err)
		}
	}
}

func importItemCategories(tx *pop.Connection, in []LegacyItemCategory) {
	for _, category := range in {
		categoryID := stringToInt(category.Id, "ItemCategory ID")

		desc := fmt.Sprintf("ItemCategory[%d].", categoryID)
		riskCategoryUUID := getRiskCategoryUUID(category.RiskCategoryId)
		newItemCategory := models.ItemCategory{
			RiskCategoryID: riskCategoryUUID,
			Name:           trim(category.Name),
			HelpText:       trim(category.HelpText),
			Status:         getItemCategoryStatus(category),
			AutoApproveMax: fixedPointStringToInt(category.AutoApproveMax, "ItemCategory.AutoApproveMax"),
			LegacyID:       nulls.NewInt(categoryID),
			CreatedAt:      parseStringTime(category.CreatedAt, desc+"CreatedAt"),
		}

		if err := newItemCategory.Create(tx); err != nil {
			log.Fatalf("failed to create item category, %s\n%+v", err, newItemCategory)
		}

		itemCategoryIDMap[categoryID] = newItemCategory.ID
		riskCategoryMap[categoryID] = riskCategoryUUID

		if err := tx.RawQuery("update item_categories set updated_at = ? where id = ?",
			parseStringTime(category.UpdatedAt, desc+"UpdatedAt"), newItemCategory.ID).Exec(); err != nil {
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
	var nPolicies, nClaims, nItems, nClaimItems, nDuplicatePolicies int
	var nUsers importPolicyUsersResult
	householdsWithMultiplePolicies := map[string]struct{}{}

	for i := range policies {
		if err := normalizePolicy(&policies[i]); err != nil {
			log.Println(err)
			continue
		}
		p := policies[i]
		p.HouseholdId = trim(p.HouseholdId)
		p.Notes = trim(p.Notes)

		var policyUUID uuid.UUID

		entityCodeID := getEntityCodeID(tx, p.EntityCode)
		policyID := stringToInt(p.Id, "Policy ID")

		if id, ok := householdPolicyMap[p.HouseholdId]; ok && p.Type == "household" {
			policyUUID = id
			policyIDMap[policyID] = policyUUID
			nDuplicatePolicies++
			householdsWithMultiplePolicies[p.HouseholdId] = struct{}{}
			appendNotesToPolicy(tx, policyUUID, p.Notes, policyID)
		} else {
			householdID := nulls.String{}
			if p.HouseholdId != "" {
				householdID = nulls.NewString(p.HouseholdId)
			}

			desc := fmt.Sprintf("Policy[%d] ", policyID)
			newPolicy := models.Policy{
				Type:         getPolicyType(p),
				HouseholdID:  householdID,
				CostCenter:   trim(p.CostCenter),
				Account:      strconv.Itoa(p.Account),
				EntityCodeID: entityCodeID,
				Notes:        p.Notes,
				LegacyID:     nulls.NewInt(policyID),
				CreatedAt:    parseStringTime(p.CreatedAt, desc+"CreatedAt"),
			}
			if newPolicy.Type == api.PolicyTypeHousehold {
				newPolicy.Account = ""
			}
			if err := newPolicy.Create(tx); err != nil {
				log.Fatalf("failed to create policy, %s\n%+v", err, newPolicy)
			}
			policyUUID = newPolicy.ID
			householdPolicyMap[p.HouseholdId] = policyUUID

			if err := tx.RawQuery("update policies set updated_at = ? where id = ?",
				parseNullStringTimeToTime(p.UpdatedAt, desc+"UpdatedAt"), newPolicy.ID).Exec(); err != nil {
				log.Fatalf("failed to set updated_at on policies, %s", err)
			}
			policyNotes[id] = p.Notes

			policyIDMap[policyID] = policyUUID

			nPolicies++
		}

		policyIsActive := importItems(tx, policyUUID, policyID, p.Items)
		nItems += len(p.Items)

		nClaimItems += importClaims(tx, policyUUID, p.Claims)
		nClaims += len(p.Claims)

		r := importPolicyUsers(tx, p, policyUUID)
		nUsers.policyUsersCreated += r.policyUsersCreated
		nUsers.usersCreated += r.usersCreated

		if policyIsActive && entityCodeID.Valid {
			activeEntities[entityCodeID.UUID] = struct{}{}
		}
	}
	setEntitiesActive(tx)

	fmt.Println("imported: ")
	fmt.Printf("  Policies: %d, Duplicate Household Policies: %d, Households with multiple Policies: %d\n",
		nPolicies, nDuplicatePolicies, len(householdsWithMultiplePolicies))
	fmt.Printf("  Claims: %d\n", nClaims)
	fmt.Printf("  Items: %d\n", nItems)
	fmt.Printf("  ClaimItems: %d\n", nClaimItems)
	fmt.Printf("  PolicyUsers: %d w/staffID: %d\n", nUsers.policyUsersCreated, nPolicyUsersWithStaffID)
	fmt.Printf("  Users: %d\n", nUsers.usersCreated)
	fmt.Printf("  Entity Codes: %d total, %d active\n", len(entityCodesMap), len(activeEntities))
}

func setEntitiesActive(tx *pop.Connection) {
	e := models.EntityCode{Active: true}
	for id := range activeEntities {
		e.ID = id
		if err := tx.UpdateColumns(&e, "active"); err != nil {
			panic("error setting entity code active " + err.Error())
		}
	}
}

func getEntityCodeID(tx *pop.Connection, code nulls.String) nulls.UUID {
	if !code.Valid {
		return nulls.UUID{}
	}
	if foundID, ok := entityCodesMap[code.String]; ok {
		return nulls.NewUUID(foundID)
	}
	entityCodeUUID := importEntityCode(tx, code.String)
	entityCodesMap[code.String] = entityCodeUUID
	return nulls.NewUUID(entityCodeUUID)
}

func importEntityCode(tx *pop.Connection, code string) uuid.UUID {
	code = trim(code)
	newEntityCode := models.EntityCode{
		Code:   code,
		Name:   code,
		Active: false,
	}
	if err := newEntityCode.Create(tx); err != nil {
		log.Fatalf("failed to create entity code, %s\n%+v", err, newEntityCode)
	}
	return newEntityCode.ID
}

func appendNotesToPolicy(tx *pop.Connection, policyUUID uuid.UUID, newNotes string, legacyID int) {
	newNotes = fmt.Sprintf("%s (ID=%d)", newNotes, legacyID)

	if policyNotes[policyUUID] != "" {
		policyNotes[policyUUID] += " ---- " + newNotes
	} else {
		policyNotes[policyUUID] = newNotes
	}

	err := tx.RawQuery("update policies set notes = ? where id = ?", policyNotes[policyUUID], policyUUID).Exec()
	if err != nil {
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
		policyType = api.PolicyTypeCorporate
	default:
		log.Fatalf("no policy type in policy '" + p.Id + "'")
	}

	return policyType
}

// normalizePolicy adjusts policy fields to pass validation checks
func normalizePolicy(p *LegacyPolicy) error {
	if p.Type == "household" {
		p.CostCenter = ""
		p.EntityCode = nulls.String{}

		if p.HouseholdId == "" {
			return fmt.Errorf("Policy[%s] HouseholdId is empty, dropping policy", p.Id)
		}
	}

	if p.Type == "ou" || p.Type == "corporate" {
		p.HouseholdId = ""

		if !p.EntityCode.Valid || p.EntityCode.String == "" {
			return fmt.Errorf("Policy[%s] EntityCode is empty, dropping policy", p.Id)
		}
		if p.CostCenter == "" {
			return fmt.Errorf("Policy[%s] CostCenter is empty, dropping policy", p.Id)
		}
	}

	return nil
}

type importPolicyUsersResult struct {
	policyUsersCreated int
	usersCreated       int
}

func importPolicyUsers(tx *pop.Connection, p LegacyPolicy, policyID uuid.UUID) importPolicyUsersResult {
	var result importPolicyUsersResult

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
		r := createPolicyUser(tx, validEmail, policyID)
		result.policyUsersCreated += r.policyUsersCreated
		result.usersCreated += r.usersCreated
	}
	return result
}

func createPolicyUser(tx *pop.Connection, email string, policyID uuid.UUID) importPolicyUsersResult {
	result := importPolicyUsersResult{}

	email = strings.ToLower(email)
	userID, ok := userEmailMap[email]
	if !ok {
		user := createUserFromEmailAddress(tx, email)
		userID = user.ID
		result.usersCreated = 1
	}

	if _, ok = policyUserMap[policyID.String()+userID.String()]; ok {
		return result
	}

	policyUser := models.PolicyUser{
		PolicyID: policyID,
		UserID:   userID,
	}
	if err := policyUser.Create(tx); err != nil {
		log.Fatalf("failed to create new PolicyUser, %s", err)
	}
	result.policyUsersCreated = 1

	policyUserMap[policyID.String()+userID.String()] = struct{}{}

	return result
}

func createUserFromEmailAddress(tx *pop.Connection, email string) models.User {
	staffID, ok := userEmailStaffIDMap[email]
	if ok {
		nPolicyUsersWithStaffID++
	}

	user := models.User{
		Email:   email,
		StaffID: staffID,
		AppRole: models.AppRoleUser,
	}
	if err := user.Create(tx); err != nil {
		log.Fatalf("failed to create new User for policy, %s", err)
	}
	userEmailMap[email] = user.ID
	return user
}

func validMailAddress(address string) (string, bool) {
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
			LegacyID:            nulls.NewInt(claimID),
			PolicyID:            policyID,
			IncidentDate:        parseStringTime(c.IncidentDate, claimDesc+"IncidentDate"),
			IncidentType:        getIncidentType(c),
			IncidentDescription: getIncidentDescription(c),
			Status:              getClaimStatus(c),
			ReviewDate:          nulls.NewTime(parseStringTime(c.ReviewDate, claimDesc+"ReviewDate")),
			ReviewerID:          getAdminUserUUID(strconv.Itoa(c.ReviewerId), claimDesc+"ReviewerID"),
			PaymentDate:         nulls.NewTime(parseStringTime(c.PaymentDate, claimDesc+"PaymentDate")),
			TotalPayout:         fixedPointStringToCurrency(c.TotalPayout, "Claim.TotalPayout"),
			CreatedAt:           parseStringTime(c.CreatedAt, claimDesc+"CreatedAt"),
		}
		if err := newClaim.Create(tx); err != nil {
			log.Fatalf("failed to create claim, %s\n%+v", err, newClaim)
		}

		if err := tx.RawQuery("update claims set updated_at = ? where id = ?",
			parseStringTime(c.UpdatedAt, claimDesc+"UpdatedAt"), newClaim.ID).Exec(); err != nil {
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
			ClaimID:         claim.ID,
			ItemID:          itemUUID,
			Status:          getClaimItemStatus(c.Status),
			IsRepairable:    getIsRepairable(c),
			RepairEstimate:  fixedPointStringToInt(c.RepairEstimate, "ClaimItem.RepairEstimate"),
			RepairActual:    fixedPointStringToInt(c.RepairActual, "ClaimItem.RepairActual"),
			ReplaceEstimate: fixedPointStringToInt(c.ReplaceEstimate, "ClaimItem.ReplaceEstimate"),
			ReplaceActual:   fixedPointStringToInt(c.ReplaceActual, "ClaimItem.ReplaceActual"),
			PayoutOption:    getPayoutOption(c.PayoutOption, itemDesc+"PayoutOption"),
			PayoutAmount:    fixedPointStringToInt(c.PayoutAmount, "ClaimItem.PayoutAmount"),
			FMV:             fixedPointStringToInt(c.Fmv, "ClaimItem.FMV"),
			ReviewDate:      parseStringTimeToNullTime(c.ReviewDate, itemDesc+"ReviewDate"),
			ReviewerID:      getAdminUserUUID(strconv.Itoa(c.ReviewerId), itemDesc+"ReviewerID"),
			LegacyID:        nulls.NewInt(claimItemID),
			CreatedAt:       parseStringTime(c.CreatedAt, itemDesc+"CreatedAt"),
		}

		if err := newClaimItem.Create(tx); err != nil {
			log.Fatalf("failed to create claim item %d, %s\nClaimItem:\n%+v", claimItemID, err, newClaimItem)
		}

		if err := tx.RawQuery("update claim_items set updated_at = ? where id = ?",
			parseStringTime(c.UpdatedAt, itemDesc+"UpdatedAt"), newClaimItem.ID).Exec(); err != nil {
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

func getClaimItemStatus(status string) api.ClaimItemStatus {
	var s api.ClaimItemStatus

	switch status {
	case "pending":
		s = api.ClaimItemStatusReview1
	case "revision":
		s = api.ClaimItemStatusRevision
	case "approved":
		s = api.ClaimItemStatusApproved
	case "denied":
		s = api.ClaimItemStatusDenied
	default:
		log.Printf("unrecognized claim item status: %s\n", status)
		s = api.ClaimItemStatus(status)
	}

	return s
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

func getIncidentType(claim LegacyClaim) api.ClaimIncidentType {
	var incidentType api.ClaimIncidentType

	switch claim.IncidentType {
	case "Broken", "Dropped":
		incidentType = api.ClaimIncidentTypeImpact
	case "Lightning", "Lightening":
		incidentType = api.ClaimIncidentTypeElectricalSurge
	case "Theft":
		incidentType = api.ClaimIncidentTypeTheft
	case "Water Damage":
		incidentType = api.ClaimIncidentTypeWaterDamage
	case "Fire", "Miscellaneous", "Unknown", "Vandalism", "War":
		incidentType = api.ClaimIncidentTypeOther
	default:
		log.Printf("unrecognized incident type: %s\n", claim.IncidentType)
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

func getClaimStatus(claim LegacyClaim) api.ClaimStatus {
	var claimStatus api.ClaimStatus

	switch claim.Status {
	case "approved":
		claimStatus = api.ClaimStatusApproved

	default:
		log.Printf("unrecognized claim status %s\n", claim.Status)
		claimStatus = api.ClaimStatus(claim.Status)
	}

	return claimStatus
}

// importItems loads legacy items onto a policy. Returns true if at least one item is approved.
func importItems(tx *pop.Connection, policyUUID uuid.UUID, policyID int, items []LegacyItem) bool {
	active := false
	for _, item := range items {
		itemID := stringToInt(item.Id, "Item ID")
		itemDesc := fmt.Sprintf("Policy[%d] Item[%d] ", policyID, itemID)

		newItem := models.Item{
			Name:              trim(item.Name),
			CategoryID:        itemCategoryIDMap[item.CategoryId],
			RiskCategoryID:    riskCategoryMap[item.CategoryId],
			InStorage:         false,
			Country:           trim(item.Country),
			Description:       trim(item.Description),
			PolicyID:          policyUUID,
			Make:              trim(item.Make),
			Model:             trim(item.Model),
			SerialNumber:      trim(item.SerialNumber),
			CoverageAmount:    fixedPointStringToInt(item.CoverageAmount, itemDesc+"CoverageAmount"),
			PurchaseDate:      parseStringTime(item.PurchaseDate, itemDesc+"PurchaseDate"),
			CoverageStatus:    getCoverageStatus(item),
			CoverageStartDate: parseStringTime(item.CoverageStartDate, itemDesc+"CoverageStartDate"),
			LegacyID:          nulls.NewInt(itemID),
			CreatedAt:         parseStringTime(item.CreatedAt, itemDesc+"CreatedAt"),
		}
		if err := newItem.Create(tx); err != nil {
			log.Fatalf("failed to create item, %s\n%+v", err, newItem)
		}
		itemIDMap[itemID] = newItem.ID

		if err := tx.RawQuery("update items set updated_at = ? where id = ?",
			parseStringTime(item.CreatedAt, itemDesc+"CreatedAt"), newItem.ID).Exec(); err != nil {
			log.Fatalf("failed to set updated_at on item, %s", err)
		}

		if newItem.CoverageStatus == api.ItemCoverageStatusApproved {
			active = true
		}
	}
	return active
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

		var entityID nulls.UUID
		if u, ok := entityCodesMap[e.Entity]; ok {
			entityID = nulls.NewUUID(u)
		}
		policyUUID, err := getPolicyUUID(e.PolicyID)
		if err != nil {
			badPolicyIDs[e.PolicyID] = struct{}{}
			// fmt.Printf("ledger has bad policy ID, %s\n", err)
			continue
		}
		l := models.LedgerEntry{
			PolicyID:           policyUUID,
			EntityCodeID:       entityID,
			Amount:             int(e.CustJE * domain.CurrencyFactor),
			DateSubmitted:      parseStringTime(e.DateSubm, "LedgerEntry.DateSubmitted"),
			DateEntered:        parseStringTimeToNullTime(e.DateEntd, "LedgerEntry.DateEntered"),
			LegacyID:           nulls.NewInt(stringToInt(e.JERecNum, "LedgerEntry.LegacyID")),
			RecordType:         nulls.NewInt(e.JERecType),
			PolicyType:         nulls.NewInt(e.PolicyType),
			AccountNumber:      strconv.Itoa(e.AccNum),
			AccountCostCenter1: e.AccCostCtr1,
			AccountCostCenter2: e.AccCostCtr2,
			EntityCode:         e.Entity,
			FirstName:          e.FirstName,
			LastName:           e.LastName,
		}
		if err := l.Create(tx); err != nil {
			log.Fatalf("failed to create ledger entry, %s\n%+v", err, l)
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

func getPolicyUUID(id int) (uuid.UUID, error) {
	if u, ok := policyIDMap[id]; ok {
		return u, nil
	}
	return uuid.UUID{}, fmt.Errorf("bad policy ID %d", id)
}

func formatDate(d string) string {
	if d == "" {
		return EmptyTime
	}
	return d
}

func parseStringTime(t, desc string) time.Time {
	if t == "" {
		if !SilenceEmptyTimeWarnings {
			log.Printf("%s is empty, using %s", desc, EmptyTime)
		}
		return emptyTime
	}
	tt, err := time.Parse(TimeFormat, t)
	if err != nil {
		log.Fatalf("failed to parse '%s' time '%s'", desc, t)
	}
	return tt
}

func parseNullStringTimeToTime(t nulls.String, desc string) time.Time {
	var tt time.Time

	if !t.Valid {
		if !SilenceEmptyTimeWarnings {
			log.Printf("%s is null, using %s", desc, EmptyTime)
		}
		return tt
	}

	var err error
	tt, err = time.Parse(TimeFormat, t.String)
	if err != nil {
		log.Fatalf("failed to parse '%s' time '%s'", desc, t.String)
	}

	return tt
}

func parseStringTimeToNullTime(t, desc string) nulls.Time {
	if t == "" {
		if !SilenceEmptyTimeWarnings {
			log.Printf("time is empty, using null time, in %s", desc)
		}
		return nulls.NewTime(emptyTime)
	}

	var tt time.Time
	var err error
	tt, err = time.Parse(TimeFormat, t)
	if err != nil {
		log.Fatalf("failed to parse '%s' time '%s'", desc, t)
	}

	return nulls.NewTime(tt)
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
