package result

import (
	"encoding/json"
	"testing"
)

func TestEventJSONIsStable(t *testing.T) {
	event := Event{
		ResultVersion: Version, Status: "breach", ContractID: "orders",
		ContractVersion: "1", RuleID: "email-valid", RuleVersion: "1", Line: 2,
		RecordID: "2", Path: "$.email", Severity: "error", Action: "quarantine",
		Message: "value must match format \"email\"", Value: "bad-email",
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"result_version":"1.0","status":"breach","contract_id":"orders","contract_version":"1","rule_id":"email-valid","rule_version":"1","line":2,"record_id":"2","path":"$.email","severity":"error","action":"quarantine","message":"value must match format \"email\"","value":"bad-email"}`
	if string(data) != want {
		t.Fatalf("event JSON = %s, want %s", data, want)
	}
}

func TestEventValidation(t *testing.T) {
	event := Event{
		ResultVersion: Version, Status: "warning", ContractID: "orders",
		ContractVersion: "1", RuleID: "email-valid", RuleVersion: "1", Line: 1,
		Path: "$.email", Severity: "warning", Action: "warn", Message: "bad",
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestExitCodes(t *testing.T) {
	if Cleared != 0 || Breach != 1 || InvalidContract != 2 || RuntimeFailure != 3 || PolicyFailure != 4 {
		t.Fatalf("unexpected exit codes: %d %d %d %d %d", Cleared, Breach, InvalidContract, RuntimeFailure, PolicyFailure)
	}
}
