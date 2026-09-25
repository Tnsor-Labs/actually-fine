package result

import "fmt"

const Version = "1.0"

type ExitCode int

const (
	Cleared         ExitCode = 0
	Breach          ExitCode = 1
	InvalidContract ExitCode = 2
	RuntimeFailure  ExitCode = 3
	PolicyFailure   ExitCode = 4
)

// Event is one machine-readable finding emitted by an execution.
// A successful record produces no event; warnings and breaches do.
type Event struct {
	ResultVersion   string `json:"result_version"`
	Status          string `json:"status"`
	ContractID      string `json:"contract_id"`
	ContractVersion string `json:"contract_version"`
	RuleID          string `json:"rule_id"`
	RuleVersion     string `json:"rule_version"`
	Line            int    `json:"line"`
	RecordID        any    `json:"record_id,omitempty"`
	Path            string `json:"path"`
	Severity        string `json:"severity"`
	Action          string `json:"action"`
	Message         string `json:"message"`
	SuggestedFix    string `json:"suggested_fix,omitempty"`
	Value           any    `json:"value,omitempty"`
}

func (e Event) Validate() error {
	if e.ResultVersion != Version {
		return fmt.Errorf("unsupported result_version %q, expected %q", e.ResultVersion, Version)
	}
	if e.Status != "warning" && e.Status != "breach" {
		return fmt.Errorf("status must be warning or breach")
	}
	if e.ContractID == "" || e.ContractVersion == "" || e.RuleID == "" || e.RuleVersion == "" {
		return fmt.Errorf("contract and rule identities are required")
	}
	if e.Line < 1 {
		return fmt.Errorf("line must be positive")
	}
	if e.Path == "" || e.Severity == "" || e.Action == "" || e.Message == "" {
		return fmt.Errorf("path, severity, action, and message are required")
	}
	return nil
}
