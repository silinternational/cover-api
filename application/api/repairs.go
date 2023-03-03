package api

const RepairTypeRenewal = "renewal"

// swagger:model
type RepairRunInput struct {
	RepairType string `json:"repair_type"`
	Date       string `json:"date"`
}

// swagger:model
type RepairResult struct {
	RepairType string `json:"repair_type"`
	Items      Items  `json:"items"`
}
