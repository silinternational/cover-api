package grifts

import (
	"github.com/gobuffalo/nulls"
)

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
	CreatedAt     string `json:"created_at"`
	LastName      string `json:"last_name"`
	FirstName     string `json:"first_name"`
	UpdatedAt     string `json:"updated_at"`
	Id            string `json:"id"`
	Email         string `json:"email"`
	EmailOverride string `json:"email_override"`
	IsBlocked     int    `json:"is_blocked"`
	LastLoginUtc  string `json:"last_login_utc"`
	StaffId       string `json:"staff_id"`
}

type LegacyPolicy struct {
	Id            string        `json:"id"`
	Claims        []LegacyClaim `json:"claims"`
	Notes         string        `json:"notes"`
	IdentCode     string        `json:"ident_code"`
	CostCenter    string        `json:"cost_center"`
	AccountDetail string        `json:"account_detail"`
	Items         []LegacyItem  `json:"items"`
	Account       int           `json:"account"`
	HouseholdId   string        `json:"household_id"`
	EntityCode    nulls.String  `json:"entity_code"`
	Type          string        `json:"type"`
	Email         string        `json:"email"`
	LastName      string        `json:"last_name"`
	FirstName     string        `json:"first_name"`
	UpdatedAt     nulls.String  `json:"updated_at"`
	CreatedAt     string        `json:"created_at"`
}

type LegacyItem struct {
	PolicyId          int          `json:"policy_id"`
	Name              string       `json:"name"`
	CoverageStartDate string       `json:"coverage_start_date"`
	CoverageEndDate   string       `json:"coverage_end_date"`
	Make              string       `json:"make"`
	Description       string       `json:"description"`
	SerialNumber      string       `json:"serial_number"`
	CreatedAt         string       `json:"created_at"`
	Id                string       `json:"id"`
	Model             string       `json:"model"`
	CategoryId        int          `json:"category_id"`
	CoverageAmount    string       `json:"coverage_amount"`
	UpdatedAt         nulls.String `json:"updated_at"`
	CoverageStatus    string       `json:"coverage_status"`
	City              string       `json:"city"`
	Country           string       `json:"country"`
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
	PolicyId            int               `json:"policy_id"`
	ReviewerId          int               `json:"reviewer_id"`
	ClaimItems          []LegacyClaimItem `json:"claim_items"`
	CreatedAt           string            `json:"created_at"`
	ReviewDate          string            `json:"review_date"`
	UpdatedAt           string            `json:"updated_at"`
	IncidentDescription string            `json:"event_description"`
	Id                  string            `json:"id"`
	IncidentType        string            `json:"event_type"`
	PaymentDate         string            `json:"payment_date"`
	TotalPayout         string            `json:"total_payout"`
	IncidentDate        string            `json:"event_date"`
	Status              string            `json:"status"`
	City                string            `json:"city"`
	Country             string            `json:"country"`
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
	City            string `json:"city"`
	Country         string `json:"country"`
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
	JERecNum    string  `json:"JE_Rec_Num"`
	DateSubm    string  `json:"Date_Subm"`
	JERecType   int     `json:"JE_Rec_Type"`
	DateEntd    string  `json:"Date_Entd"`
	AccCostCtr2 string  `json:"Acc_CostCtr2"`
	FirstName   string  `json:"First_Name"`
	LastName    string  `json:"Last_Name"`
	AccNum      int     `json:"Acc_Num"`
	Field1      string  `json:"Field1"`
	AccCostCtr1 string  `json:"Acc_CostCtr1"`
	PolicyID    int     `json:"Policy_ID"`
	Entity      string  `json:"Entity"`
	PolicyType  int     `json:"Policy_Type"`
	CustJE      float64 `json:"Cust_JE"`
	RMJE        float64 `json:"RM_JE"`
}
