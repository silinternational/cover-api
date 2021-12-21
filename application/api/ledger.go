package api

// swagger:model
type BatchApproveResponse struct {
	NumberOfRecordsApproved int `json:"number_of_records_approved"`
}

// swagger:model
type LedgerReconcileInput struct {
	EndDate string `json:"end_date"`
}
