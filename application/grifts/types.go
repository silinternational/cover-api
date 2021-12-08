package grifts

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gobuffalo/nulls"
)

// time types that use MySQL format for deserialization
type (
	MySQLTime     time.Time
	MySQLNullTime nulls.Time
)

func (t *MySQLTime) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	t1, err := time.Parse(MySQLTimeFormat, v)
	*t = MySQLTime(t1)

	return err
}

func (t *MySQLNullTime) UnmarshalJSON(data []byte) error {
	t.Valid = false
	if string(data) == "null" || string(data) == "" {
		return nil
	}

	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	t1, err := time.Parse(MySQLTimeFormat, v)
	if err == nil && !t1.IsZero() {
		t.Time = t1
		t.Valid = true
		return nil
	}

	if err != nil {
		return fmt.Errorf("MySQLNullTime unmarshal error %w", err)
	}
	return nil
}

type LegacyData struct {
	Users          []LegacyUser         `json:"users"`
	Policies       []LegacyPolicy       `json:"policies"`
	PolicyTypes    []PolicyType         `json:"PolicyType"`
	Maintenance    []Maintenance        `json:"Maintenance"`
	JournalEntries []JournalEntry       `json:"tblJEntry"`
	ItemCategories []LegacyItemCategory `json:"item_categories"`
	RiskCategories []LegacyRiskCategory `json:"risk_categories"`
	LossReasons    []LossReason         `json:"LossReason"`
}

type LegacyUser struct {
	LastName      string    `json:"last_name"`
	FirstName     string    `json:"first_name"`
	Id            string    `json:"id"`
	Email         string    `json:"email"`
	EmailOverride string    `json:"email_override"`
	IsBlocked     int       `json:"is_blocked"`
	StaffId       string    `json:"staff_id"`
	LastLoginUtc  MySQLTime `json:"last_login_utc"`
	CreatedAt     MySQLTime `json:"created_at"`
	UpdatedAt     MySQLTime `json:"updated_at"`
}

type LegacyPolicy struct {
	Id            string        `json:"id"`
	FirstName     string        `json:"first_name"`
	LastName      string        `json:"last_name"`
	Email         string        `json:"email"`
	IdentCode     string        `json:"ident_code"`
	Type          string        `json:"type"`
	HouseholdId   string        `json:"household_id"`
	CostCenter    string        `json:"cost_center"`
	Account       int           `json:"account"`
	AccountDetail string        `json:"account_detail"`
	EntityCode    nulls.String  `json:"entity_code"`
	Notes         string        `json:"notes"`
	UpdatedAt     MySQLTime     `json:"updated_at"`
	CreatedAt     MySQLTime     `json:"created_at"`
	Claims        []LegacyClaim `json:"claims"`
	Items         []LegacyItem  `json:"items"`
}

type LegacyItem struct {
	Id              string        `json:"id"`
	PolicyId        int           `json:"policy_id"`
	Name            string        `json:"name"`
	CoverageEndDate MySQLNullTime `json:"coverage_end_date"`
	Make            string        `json:"make"`
	Description     string        `json:"description"`
	SerialNumber    string        `json:"serial_number"`
	Model           string        `json:"model"`
	CategoryId      int           `json:"category_id"`
	CoverageAmount  string        `json:"coverage_amount"`
	CoverageStatus  string        `json:"coverage_status"`
	City            string        `json:"city"`
	Country         string        `json:"country"`
	CreatedAt       MySQLTime     `json:"created_at"`
	UpdatedAt       MySQLTime     `json:"updated_at"`
}

type LegacyItemCategory struct {
	Id             string    `json:"id"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	AutoApproveMax string    `json:"auto_approve_max"`
	RiskCategoryId int       `json:"risk_category_id"`
	HelpText       string    `json:"help_text"`
	CreatedAt      MySQLTime `json:"created_at"`
	UpdatedAt      MySQLTime `json:"updated_at"`
}

type LegacyClaim struct {
	Id                  string            `json:"id"`
	PolicyId            int               `json:"policy_id"`
	ReviewerId          int               `json:"reviewer_id"`
	ReviewDate          MySQLNullTime     `json:"review_date"`
	IncidentDescription string            `json:"event_description"`
	IncidentType        string            `json:"event_type"`
	PaymentDate         MySQLNullTime     `json:"payment_date"`
	TotalPayout         string            `json:"total_payout"`
	IncidentDate        MySQLTime         `json:"event_date"`
	City                string            `json:"city"`
	Country             string            `json:"country"`
	CreatedAt           MySQLTime         `json:"created_at"`
	UpdatedAt           MySQLTime         `json:"updated_at"`
	ClaimItems          []LegacyClaimItem `json:"claim_items"`
}

type LegacyClaimItem struct {
	Id              string        `json:"id"`
	ReviewerId      int           `json:"reviewer_id"`
	PayoutAmount    string        `json:"payout_amount"`
	CoverageAmount  string        `json:"covered_value"`
	ItemId          int           `json:"item_id"`
	PayoutOption    string        `json:"payout_option"`
	RepairEstimate  string        `json:"repair_estimate"`
	ReviewDate      MySQLNullTime `json:"review_date"`
	City            string        `json:"city"`
	Country         string        `json:"country"`
	ReplaceActual   string        `json:"replace_actual"`
	ReplaceEstimate string        `json:"replace_estimate"`
	IsRepairable    int           `json:"is_repairable"`
	RepairActual    string        `json:"repair_actual"`
	Fmv             string        `json:"fmv"`
	ClaimId         int           `json:"claim_id"`
	CreatedAt       MySQLTime     `json:"created_at"`
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
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	PolicyMax nulls.Int `json:"policy_max"`
	CreatedAt MySQLTime `json:"created_at"`
	UpdatedAt MySQLTime `json:"updated_at"`
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
	JERecNum    string        `json:"JE_Rec_Num"`
	DateSubm    MySQLNullTime `json:"Date_Subm"`
	JERecType   int           `json:"JE_Rec_Type"`
	DateEntd    MySQLNullTime `json:"Date_Entd"`
	AccCostCtr2 string        `json:"Acc_CostCtr2"`
	FirstName   string        `json:"First_Name"`
	LastName    string        `json:"Last_Name"`
	AccNum      int           `json:"Acc_Num"`
	Field1      string        `json:"Field1"`
	AccCostCtr1 string        `json:"Acc_CostCtr1"`
	PolicyID    int           `json:"Policy_ID"`
	Entity      string        `json:"Entity"`
	PolicyType  int           `json:"Policy_Type"`
	CustJE      float64       `json:"Cust_JE"`
	RMJE        float64       `json:"RM_JE"`
}
