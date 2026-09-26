package brokoli

import (
	"bytes"
	"testing"

	"github.com/Tnsor-Labs/actually-fine/contract"
	"github.com/Tnsor-Labs/actually-fine/result"
)

func TestRunPreservesRoutingAndEvidence(t *testing.T) {
	c := contract.Contract{IRVersion: "1.0", Metadata: contract.Metadata{ID: "orders", Version: "1"}, Input: contract.Input{Kind: "record-stream"}, Rules: []contract.Rule{{ID: "required-email", Kind: "record", Path: "$.email", Predicate: contract.Predicate{Op: "required"}, OnBreach: contract.Policy{Action: "quarantine"}}}}
	var accepted, quarantined bytes.Buffer
	events := 0
	summary, err := Run(Request{Contract: c, Input: bytes.NewBufferString(`{"id":"ok","email":"a@example.com"}
{"id":"bad"}
`), Accepted: func(raw []byte) error { _, err := accepted.Write(append(raw, '\n')); return err }, Quarantined: func(raw []byte) error { _, err := quarantined.Write(append(raw, '\n')); return err }, Evidence: func(event result.Event) error { events++; return nil }, InvalidRecord: InvalidRecordQuarantine})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Cleared != 1 || summary.Quarantined != 1 || summary.ExitCode() != result.Breach {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if accepted.String() != `{"id":"ok","email":"a@example.com"}
` || quarantined.String() != `{"id":"bad"}
` || events != 1 {
		t.Fatalf("unexpected routing: accepted=%q quarantine=%q events=%d", accepted.String(), quarantined.String(), events)
	}
}
