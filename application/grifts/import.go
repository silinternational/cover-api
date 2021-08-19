package grifts

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
	"github.com/markbates/grift/grift"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

/*
	This is a temporary command-line utility to read a JSON file with data from the legacy Riskman software.

	The input file is expected to have a number of top-level objects, as defined in `TempData`. The `Policies`
	list is a complex structure contained related data. The remainder are simple objects.

TODO:
	1. Import ClaimItems
	2. Import users and assign correct IDs (claim.reviewer_id)
	3. Import policy members (parse email field on polices)
	4. Import other tables (e.g. Journal Entries)
	5. Convert panic to log.Fatal or log.Fatalf
*/

const TimeFormat = "2006-01-02 15:04:05"

type LegacyData struct {
	Policies       []LegacyPolicy       `json:"policies"`
	PolicyTypes    []PolicyType         `json:"PolicyType"`
	Maintenance    []Maintenance        `json:"Maintenance"`
	JournalEntries []JournalEntry       `json:"tblJEntry"`
	ItemCategories []LegacyItemCategory `json:"item_categories"`
	RiskCategories []LegacyRiskCategory `json:"risk_categories"`
	LossReasons    []LossReason         `json:"LossReason"`
}

type LegacyPolicy struct {
	Id          string        `json:"id"`
	Claims      []LegacyClaim `json:"claims"`
	Notes       string        `json:"notes"`
	IdentCode   string        `json:"ident_code"`
	CostCenter  string        `json:"cost_center"`
	Items       []LegacyItem  `json:"items"`
	Account     int           `json:"account"`
	HouseholdId string        `json:"household_id"`
	EntityCode  nulls.String  `json:"entity_code"`
	Type        string        `json:"type"`
	UpdatedAt   nulls.String  `json:"updated_at"`
	CreatedAt   string        `json:"created_at"`
}

type LegacyItem struct {
	PolicyId          int          `json:"policy_id"`
	InStorage         int          `json:"in_storage"`
	PurchaseDate      string       `json:"purchase_date"`
	Name              string       `json:"name"`
	CoverageStartDate string       `json:"coverage_start_date"`
	Make              string       `json:"make"`
	Description       string       `json:"description"`
	SerialNumber      string       `json:"serial_number"`
	CreatedAt         string       `json:"created_at"`
	Id                string       `json:"id"`
	Country           string       `json:"country"`
	Model             string       `json:"model"`
	CategoryId        int          `json:"category_id"`
	CoverageAmount    string       `json:"coverage_amount"`
	UpdatedAt         nulls.String `json:"updated_at"`
	PolicyDependentId int          `json:"policy_dependent_id"`
	CoverageStatus    string       `json:"coverage_status"`
}

type LegacyItemCategory struct {
	CreatedAt      string `json:"created_at"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	UpdatedAt      string `json:"updated_at"`
	AutoApproveMax string `json:"auto_approve_max"`
	RiskCategoryId int    `json:"risk_category_id"`
	HelpText       string `json:"help_text"`
	Id             string `json:"id"`
}

type LegacyClaim struct {
	PolicyId         int               `json:"policy_id"`
	ReviewerId       int               `json:"reviewer_id"`
	ClaimItems       []LegacyClaimItem `json:"claim_items"`
	CreatedAt        string            `json:"created_at"`
	ReviewDate       string            `json:"review_date"`
	UpdatedAt        string            `json:"updated_at"`
	EventDescription string            `json:"event_description"`
	Id               string            `json:"id"`
	EventType        string            `json:"event_type"`
	PaymentDate      string            `json:"payment_date"`
	TotalPayout      string            `json:"total_payout"`
	EventDate        string            `json:"event_date"`
	Status           string            `json:"status"`
}

type LegacyClaimItem struct {
	ReviewerId      int    `json:"reviewer_id"`
	PayoutAmount    string `json:"payout_amount"`
	ItemId          int    `json:"item_id"`
	PayoutOption    string `json:"payout_option"`
	RepairEstimate  string `json:"repair_estimate"`
	UpdatedAt       string `json:"updated_at"`
	ReviewDate      string `json:"review_date"`
	CreatedAt       string `json:"created_at"`
	Location        string `json:"location"`
	ReplaceActual   string `json:"replace_actual"`
	Id              string `json:"id"`
	ReplaceEstimate string `json:"replace_estimate"`
	IsRepairable    int    `json:"is_repairable"`
	Status          string `json:"status"`
	RepairActual    string `json:"repair_actual"`
	Fmv             string `json:"fmv"`
	ClaimId         int    `json:"claim_id"`
}

type PolicyType struct {
	MusicMax        int     `json:"Music_Max"`
	MinRefund       int     `json:"Min_Refund"`
	PolTypeRecNum   string  `json:"PolType_Rec_Num"`
	WarLimit        float64 `json:"War_Limit"`
	PolicyDeductMin int     `json:"Policy_Deduct_Min"`
	PolicyRate      float64 `json:"Policy_Rate"`
	PolicyType      string  `json:"Policy_Type"`
	MinFee          int     `json:"Min_Fee"`
	PolicyDeductPct float64 `json:"Policy_Deduct_Pct"`
}

type LegacyRiskCategory struct {
	CreatedAt string       `json:"created_at"`
	Name      string       `json:"name"`
	PolicyMax nulls.Int    `json:"policy_max"`
	UpdatedAt nulls.String `json:"updated_at"`
	Id        string       `json:"id"`
}

type LossReason struct {
	ReasonRecNum string `json:"Reason_Rec_Num"`
	Reason       string `json:"Reason"`
}

type Maintenance struct {
	DateResolved nulls.String `json:"dateResolved"`
	Seq          string       `json:"seq"`
	Problem      string       `json:"problem"`
	DateReported string       `json:"dateReported"`
	Resolution   nulls.String `json:"resolution"`
}

type JournalEntry struct {
	FirstName   nulls.String `json:"First_Name"`
	JERecNum    string       `json:"JE_Rec_Num"`
	DateSubm    string       `json:"Date_Subm"`
	JERecType   int          `json:"JE_Rec_Type"`
	DateEntd    string       `json:"Date_Entd"`
	AccCostCtr2 string       `json:"Acc_CostCtr2"`
	LastName    string       `json:"Last_Name"`
	AccNum      int          `json:"Acc_Num"`
	Field1      nulls.String `json:"Field1"`
	AccCostCtr1 string       `json:"Acc_CostCtr1"`
	PolicyID    int          `json:"Policy_ID"`
	Entity      string       `json:"Entity"`
	PolicyType  int          `json:"Policy_Type"`
	CustJE      float64      `json:"Cust_JE"`
	RMJE        float64      `json:"RM_JE"`
}

// itemCategoryMap is a map of legacy ID to new ID
var itemCategoryMap = map[int]uuid.UUID{}

var _ = grift.Namespace("db", func() {
	_ = grift.Desc("import", "Import legacy data")
	_ = grift.Add("import", func(c *grift.Context) error {
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

		if err := models.DB.Transaction(func(tx *pop.Connection) error {
			fmt.Println("record counts: ")
			fmt.Printf("  Policies: %d\n", len(obj.Policies))
			fmt.Printf("  PolicyTypes: %d\n", len(obj.PolicyTypes))
			fmt.Printf("  Maintenance: %d\n", len(obj.Maintenance))
			fmt.Printf("  JournalEntries: %d\n", len(obj.JournalEntries))
			fmt.Printf("  ItemCategories: %d\n", len(obj.ItemCategories))
			fmt.Printf("  RiskCategories: %d\n", len(obj.RiskCategories))
			fmt.Printf("  LossReasons: %d\n", len(obj.LossReasons))
			fmt.Println("")

			importItemCategories(tx, obj.ItemCategories)
			importPolicies(tx, obj.Policies)

			return errors.New("blocking transaction commit until everything is ready")
		}); err != nil {
			panic("failed to import, " + err.Error())
		}

		return nil
	})
})

func importItemCategories(tx *pop.Connection, in []LegacyItemCategory) {
	fmt.Println("Item categories")
	fmt.Println("legacy_id,id,status,risk_category_id,name,auto_approve_max,help_text")

	for _, i := range in {
		categoryID := stringToInt(i.Id, "ItemCategory ID")

		newItemCategory := models.ItemCategory{
			RiskCategoryID: getRiskCategoryUUID(i.RiskCategoryId),
			Name:           i.Name,
			HelpText:       i.HelpText,
			Status:         getItemCategoryStatus(i),
			AutoApproveMax: fixedPointStringToInt(i.AutoApproveMax, "ItemCategory.AutoApproveMax"),
			CreatedAt:      parseStringTime(i.CreatedAt, "ItemCategory.CreatedAt"),
			UpdatedAt:      parseStringTime(i.UpdatedAt, "ItemCategory.UpdatedAt"),
			LegacyID:       nulls.NewInt(categoryID),
		}

		if err := newItemCategory.Create(tx); err != nil {
			panic(fmt.Sprintf("failed to create item category, %s\n%+v", err, newItemCategory))
		}

		itemCategoryMap[categoryID] = newItemCategory.ID

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
		return uuid.FromStringOrNil(models.RiskCategoryStationaryIDString)
	case 2, 3:
		return uuid.FromStringOrNil(models.RiskCategoryMobileIDString)
	}
	fmt.Printf("unrecognized risk category ID %d", legacyID)
	return uuid.FromStringOrNil(models.RiskCategoryMobileIDString)
}

func getItemCategoryStatus(itemCategory LegacyItemCategory) api.ItemCategoryStatus {
	var status api.ItemCategoryStatus

	// TODO: add other status values to this function

	switch itemCategory.Status {
	case "enabled":
		status = api.ItemCategoryStatusEnabled

	case "deprecated":
		status = api.ItemCategoryStatusDeprecated

	default:
		fmt.Printf("unrecognized item category status %s\n", itemCategory.Status)
		status = api.ItemCategoryStatus(itemCategory.Status)
	}

	return status
}

func importPolicies(tx *pop.Connection, in []LegacyPolicy) {
	nClaims := 0
	nItems := 0
	for i := range in {
		normalizePolicy(&in[i])
		p := in[i]

		newPolicy := models.Policy{
			Type:        getPolicyType(p),
			HouseholdID: p.HouseholdId,
			CostCenter:  p.CostCenter,
			Account:     strconv.Itoa(p.Account),
			EntityCode:  p.EntityCode.String,
			LegacyID:    nulls.NewInt(stringToInt(p.Id, "Policy ID")),
			CreatedAt:   parseStringTime(p.CreatedAt, "Policy.CreatedAt"),
			UpdatedAt:   parseNullStringTime(p.UpdatedAt, "Policy.UpdatedAt"),
		}
		if err := newPolicy.Create(tx); err != nil {
			panic(fmt.Sprintf("failed to create policy, %s\n%+v", err, newPolicy))
		}

		importClaims(tx, newPolicy, p.Claims)
		nClaims += len(p.Claims)

		importItems(tx, newPolicy, p.Items)
		nItems += len(p.Items)
	}

	fmt.Println("imported: ")
	fmt.Printf("  Policies: %d\n", len(in))
	fmt.Printf("  Claims: %d\n", nClaims)
	fmt.Printf("  Items: %d\n", nItems)
	fmt.Println("")
}

// getPolicyType gets the correct policy type
func getPolicyType(p LegacyPolicy) api.PolicyType {
	var policyType api.PolicyType

	switch p.Type {
	case "household":
		policyType = api.PolicyTypeHousehold
	case "ou", "corporate":
		policyType = api.PolicyTypeCorporate
	}

	return policyType
}

// normalizePolicy adjusts policy fields to pass validation checks
func normalizePolicy(p *LegacyPolicy) {
	if p.Type == "household" {
		// TODO: fix input data so this isn't needed
		if p.HouseholdId == "" {
			fmt.Printf("empty HouseholdId on Policy %s\n", p.Id)
			p.HouseholdId = "-"
		}
	}
	if p.Type == "ou" || p.Type == "corporate" {
		// TODO: fix input data so this isn't needed
		if !p.EntityCode.Valid || p.EntityCode.String == "" {
			fmt.Printf("empty EntityCode on Policy %s\n", p.Id)
			p.EntityCode = nulls.NewString("-")
		}
		if p.CostCenter == "" {
			fmt.Printf("empty CostCenter on Policy %s\n", p.Id)
			p.CostCenter = "-"
		}
	}
}

func importClaims(tx *pop.Connection, policy models.Policy, claims []LegacyClaim) {
	for _, c := range claims {
		newClaim := models.Claim{
			LegacyID:         nulls.NewInt(stringToInt(c.Id, "Claim ID")),
			PolicyID:         policy.ID,
			EventDate:        parseStringTime(c.EventDate, "EventDate"),
			EventType:        getEventType(c),
			EventDescription: getEventDescription(c),
			Status:           getClaimStatus(c),
			ReviewDate:       nulls.NewTime(parseStringTime(c.ReviewDate, "Claim.ReviewDate")),
			// TODO: need user IDs
			// ReviewerID:       c.ReviewerId,
			PaymentDate: nulls.NewTime(parseStringTime(c.PaymentDate, "Claim.PaymentDate")),
			TotalPayout: fixedPointStringToInt(c.TotalPayout, "Claim.TotalPayout"),
			CreatedAt:   parseStringTime(c.CreatedAt, "Claim.CreatedAt"),
			UpdatedAt:   parseStringTime(c.UpdatedAt, "Claim.UpdatedAt"),
		}
		if err := newClaim.Create(tx); err != nil {
			panic(fmt.Sprintf("failed to create claim, %s\n%+v", err, newClaim))
		}
	}
}

func floatToInt(f float64) int {
	// TODO: fix this so we do not lose precision
	return int(math.Round(f))
}

func getEventType(claim LegacyClaim) api.ClaimEventType {
	var eventType api.ClaimEventType

	// TODO: resolve "missing" types

	switch claim.EventType {
	case "Broken", "Dropped":
		eventType = api.ClaimEventTypeImpact
	case "Lightning", "Lightening":
		eventType = api.ClaimEventTypeElectrical
	case "Theft":
		eventType = api.ClaimEventTypeTheft
	case "Water Damage":
		eventType = api.ClaimEventTypeWater
	case "Fire", "Miscellaneous", "Unknown", "Vandalism", "War":
		eventType = api.ClaimEventTypeOther
	default:
		fmt.Printf("unrecognized event type: %s\n", claim.EventType)
		eventType = api.ClaimEventTypeOther
	}

	return eventType
}

func getEventDescription(claim LegacyClaim) string {
	if claim.EventDescription == "" {
		// TODO: provide event descriptions on source data
		// fmt.Printf("missing event description on claim %s\n", claim.Id)
		return "-"
	}
	return claim.EventDescription
}

func getClaimStatus(claim LegacyClaim) api.ClaimStatus {
	var claimStatus api.ClaimStatus

	// TODO: add other status values to this function

	switch claim.Status {
	case "approved":
		claimStatus = api.ClaimStatusApproved

	default:
		fmt.Printf("unrecognized claim status %s\n", claim.Status)
		claimStatus = api.ClaimStatus(claim.Status)
	}

	return claimStatus
}

func importItems(tx *pop.Connection, policy models.Policy, items []LegacyItem) {
	for _, item := range items {
		newItem := models.Item{
			// TODO: name/policy needs to be unique
			Name:              item.Name + domain.GetUUID().String(),
			CategoryID:        itemCategoryMap[item.CategoryId],
			InStorage:         false,
			Country:           item.Country,
			Description:       item.Description,
			PolicyID:          policy.ID,
			Make:              item.Make,
			Model:             item.Model,
			SerialNumber:      item.SerialNumber,
			CoverageAmount:    fixedPointStringToInt(item.CoverageAmount, "Item.CoverageAmount"),
			PurchaseDate:      parseStringTime(item.PurchaseDate, "Item.PurchaseDate"),
			CoverageStatus:    getCoverageStatus(item),
			CoverageStartDate: parseStringTime(item.CoverageStartDate, "Item.CoverageStartDate"),
			LegacyID:          nulls.NewInt(stringToInt(item.Id, "Item ID")),
			CreatedAt:         parseStringTime(item.CreatedAt, "Item.CreatedAt"),
			UpdatedAt:         parseNullStringTime(item.UpdatedAt, "Item.UpdatedAt"),
		}
		if err := newItem.Create(tx); err != nil {
			panic(fmt.Sprintf("failed to create item, %s\n%+v", err, newItem))
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
		fmt.Printf("unknown coverage status %s\n", item.CoverageStatus)
		coverageStatus = api.ItemCoverageStatus(item.CoverageStatus)
	}

	return coverageStatus
}

func parseStringTime(t, desc string) time.Time {
	if t == "" {
		return time.Time{}
	}
	createdAt, err := time.Parse(TimeFormat, t)
	if err != nil {
		panic(fmt.Sprintf("failed to parse '%s' time '%s'", desc, t))
	}
	return createdAt
}

func parseNullStringTime(t nulls.String, desc string) time.Time {
	var updatedAt time.Time
	if t.Valid {
		var err error
		updatedAt, err = time.Parse(TimeFormat, t.String)
		if err != nil {
			panic(fmt.Sprintf("failed to parse '%s' time '%s'", desc, t.String))
		}
	}
	return updatedAt
}

func stringToInt(s, msg string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("%s '%s' is not an int", msg, s))
	}
	return n
}

func fixedPointStringToInt(s, field string) int {
	if s == "" {
		log.Printf("%s is empty", field)
		return 0
	}

	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		log.Fatalf("%s has more than one '.' character: '%s'", field, s)
	}
	intPart := stringToInt(parts[0], field+" left of decimal")
	if len(parts[1]) != 2 {
		log.Fatalf("%s does not have two digits after the decimal: %s", field, s)
	}
	fractionalPart := stringToInt(parts[1], field+" right of decimal")
	return intPart*100 + fractionalPart
}
