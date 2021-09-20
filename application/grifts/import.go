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

TODO:
	1. Import other tables (e.g. Journal Entries)
	2. Make name/policy unique
	3. Strip extra whitespace (household ID, ?)
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

			return errors.New("blocking transaction commit until everything is ready")
		}); err != nil {
			log.Fatalf("failed to import, %s", err)
		}

		return nil
	})
})

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

func init() {
	emptyTime, _ = time.Parse(TimeFormat, EmptyTime)
	pop.Debug = false // Disable the Pop log messages
}

func importAdminUsers(tx *pop.Connection, in []LegacyUser) {
	fmt.Println("Admin Users")
	fmt.Println("id,email,email_override,first_name,last_name,last_login_utc,location,staff_id,app_role")

	for _, user := range in {
		userID := stringToInt(user.Id, "User ID")
		userDesc := fmt.Sprintf("User[%d].", userID)

		newUser := models.User{
			Email:         user.Email,
			EmailOverride: user.EmailOverride,
			FirstName:     user.FirstName,
			LastName:      user.LastName,
			LastLoginUTC:  parseStringTime(user.LastLoginUtc, userDesc+"LastLoginUTC"),
			Location:      user.Location,
			StaffID:       user.StaffId,
			AppRole:       models.AppRoleAdmin,
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

		fmt.Printf(`"%s","%s","%s","%s","%s","%s","%s","%s","%s"`+"\n",
			newUser.ID, newUser.Email, newUser.EmailOverride, newUser.FirstName, newUser.LastName,
			newUser.LastLoginUTC, newUser.Location, newUser.StaffID, newUser.AppRole,
		)
	}

	fmt.Println()
}

func importItemCategories(tx *pop.Connection, in []LegacyItemCategory) {
	fmt.Println("Item categories")
	fmt.Println("legacy_id,id,status,risk_category_id,name,auto_approve_max,help_text")

	for _, i := range in {
		categoryID := stringToInt(i.Id, "ItemCategory ID")

		desc := fmt.Sprintf("ItemCategory[%d].", categoryID)
		riskCategoryUUID := getRiskCategoryUUID(i.RiskCategoryId)
		newItemCategory := models.ItemCategory{
			RiskCategoryID: riskCategoryUUID,
			Name:           i.Name,
			HelpText:       i.HelpText,
			Status:         getItemCategoryStatus(i),
			AutoApproveMax: fixedPointStringToInt(i.AutoApproveMax, "ItemCategory.AutoApproveMax"),
			LegacyID:       nulls.NewInt(categoryID),
			CreatedAt:      parseStringTime(i.CreatedAt, desc+"CreatedAt"),
		}

		if err := newItemCategory.Create(tx); err != nil {
			log.Fatalf("failed to create item category, %s\n%+v", err, newItemCategory)
		}

		itemCategoryIDMap[categoryID] = newItemCategory.ID
		riskCategoryMap[categoryID] = riskCategoryUUID

		if err := tx.RawQuery("update item_categories set updated_at = ? where id = ?",
			parseStringTime(i.UpdatedAt, desc+"UpdatedAt"), newItemCategory.ID).Exec(); err != nil {
			log.Fatalf("failed to set updated_at on item_categories, %s", err)
		}

		fmt.Printf(`%d,"%s","%s",%s,"%s",%d,"%s"`+"\n",
			newItemCategory.LegacyID.Int, newItemCategory.ID, newItemCategory.Status,
			newItemCategory.RiskCategoryID, newItemCategory.Name, newItemCategory.AutoApproveMax,
			newItemCategory.HelpText)
	}

	fmt.Println("")
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

func importPolicies(tx *pop.Connection, in []LegacyPolicy) {
	var nPolicies, nClaims, nItems, nClaimItems, nPolicyUsers, nDuplicatePolicies int
	householdsWithMultiplePolicies := map[string]struct{}{}

	for i := range in {
		if err := normalizePolicy(&in[i]); err != nil {
			log.Println(err)
			continue
		}
		p := in[i]

		var policyUUID uuid.UUID

		if id, ok := householdPolicyMap[p.HouseholdId]; ok && p.Type == "household" {
			policyUUID = id
			nDuplicatePolicies++
			householdsWithMultiplePolicies[p.HouseholdId] = struct{}{}
			appendNotesToPolicy(tx, policyUUID, p.Notes)
			// TODO: are there other fields that should be examined to ensure we do not lose data?
		} else {
			policyID := stringToInt(p.Id, "Policy ID")
			householdID := nulls.String{}
			if p.HouseholdId != "" {
				householdID = nulls.NewString(p.HouseholdId)
			}

			desc := fmt.Sprintf("Policy[%d].", policyID)
			newPolicy := models.Policy{
				Type:         getPolicyType(p),
				HouseholdID:  householdID,
				CostCenter:   p.CostCenter,
				Account:      strconv.Itoa(p.Account),
				EntityCodeID: getEntityCodeID(tx, p.EntityCode),
				Notes:        p.Notes,
				LegacyID:     nulls.NewInt(policyID),
				CreatedAt:    parseStringTime(p.CreatedAt, desc+"CreatedAt"),
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

			nPolicies++
		}

		importItems(tx, policyUUID, p.Items)
		nItems += len(p.Items)

		nClaimItems += importClaims(tx, policyUUID, p.Claims)
		nClaims += len(p.Claims)

		nPolicyUsers += importPolicyUsers(tx, p, policyUUID)
	}

	fmt.Println("imported: ")
	fmt.Printf("  Policies: %d, Duplicate Household Policies: %d, Households with multiple Policies: %d\n",
		nPolicies, nDuplicatePolicies, len(householdsWithMultiplePolicies))
	fmt.Printf("  Claims: %d\n", nClaims)
	fmt.Printf("  Items: %d\n", nItems)
	fmt.Printf("  ClaimItems: %d\n", nClaimItems)
	fmt.Printf("  PolicyUsers: %d w/staffID: %d\n", nPolicyUsers, nPolicyUsersWithStaffID)
	fmt.Printf("  Entity Codes: %d\n", len(entityCodesMap))
	fmt.Println("")
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
	newEntityCode := models.EntityCode{
		Code: code,
		Name: code,
	}
	if err := newEntityCode.Create(tx); err != nil {
		log.Fatalf("failed to create entity code, %s\n%+v", err, newEntityCode)
	}
	return newEntityCode.ID
}

func appendNotesToPolicy(tx *pop.Connection, policyUUID uuid.UUID, newNotes string) {
	if newNotes == "" {
		return
	}

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
			return fmt.Errorf("Policy[%s].HouseholdId is empty, dropping policy", p.Id)
		}
	}

	if p.Type == "ou" || p.Type == "corporate" {
		p.HouseholdId = ""

		if !p.EntityCode.Valid || p.EntityCode.String == "" {
			return fmt.Errorf("Policy[%s].EntityCode is empty, dropping policy", p.Id)
		}
		if p.CostCenter == "" {
			return fmt.Errorf("Policy[%s].CostCenter is empty, dropping policy", p.Id)
		}
	}

	return nil
}

func importPolicyUsers(tx *pop.Connection, p LegacyPolicy, policyID uuid.UUID) int {
	if p.Email == "" {
		if !SilenceBadEmailWarning {
			log.Printf("Policy[%s].Email is empty\n", p.Id)
		}
		return 0
	}

	n := 0
	s := strings.Split(strings.ReplaceAll(p.Email, " ", ","), ",")
	for _, email := range s {
		validEmail, ok := validMailAddress(email)
		if !ok {
			if !SilenceBadEmailWarning {
				log.Printf("Policy[%s] has an invalid email address: '%s'\n", p.Id, email)
			}
			continue
		}
		createPolicyUser(tx, validEmail, policyID)
		n++
	}
	return n
}

func createPolicyUser(tx *pop.Connection, email string, policyID uuid.UUID) {
	email = strings.ToLower(email)
	userID, ok := userEmailMap[email]
	if !ok {
		user := createUserFromEmailAddress(tx, email)
		userID = user.ID
	}

	if _, ok = policyUserMap[policyID.String()+userID.String()]; ok {
		return
	}

	policyUser := models.PolicyUser{
		PolicyID: policyID,
		UserID:   userID,
	}
	if err := policyUser.Create(tx); err != nil {
		log.Fatalf("failed to create new PolicyUser, %s", err)
	}

	policyUserMap[policyID.String()+userID.String()] = struct{}{}
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
		itemDesc := fmt.Sprintf("ClaimItem[%d].", claimItemID)

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
		s = api.ClaimItemStatusPending
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

func importItems(tx *pop.Connection, policyID uuid.UUID, items []LegacyItem) {
	for _, item := range items {
		itemID := stringToInt(item.Id, "Item ID")
		itemDesc := fmt.Sprintf("Item[%d].", itemID)

		newItem := models.Item{
			// TODO: name/policy needs to be unique
			Name:              item.Name + domain.GetUUID().String(),
			CategoryID:        itemCategoryIDMap[item.CategoryId],
			RiskCategoryID:    riskCategoryMap[item.CategoryId],
			InStorage:         false,
			Country:           item.Country,
			Description:       item.Description,
			PolicyID:          policyID,
			Make:              item.Make,
			Model:             item.Model,
			SerialNumber:      item.SerialNumber,
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
	}
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
