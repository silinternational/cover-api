package api

const AuditTypeRenewal = "renewal"

// swagger:model
type AuditRunInput struct {
	AuditType string `json:"audit_type"`
	Date      string `json:"date"`
}

// swagger:model
type AuditResult struct {
	AuditType string `json:"audit_type"`
	Items     Items  `json:"items"`
}
