package engine

import (
	"testing"

	"github.com/Tnsor-Labs/actually-fine/contract"
)

func TestCheckRules(t *testing.T) {
	min := 0.0
	max := 100.0
	c := contract.Contract{
		IRVersion: "1.0",
		Metadata:  contract.Metadata{ID: "orders", Version: "1"},
		Input:     contract.Input{Kind: "record-stream"},
		Rules: []contract.Rule{
			{ID: "email", Kind: "record", Path: "$.email", Predicate: contract.Predicate{Op: "format", Format: "email"}, OnBreach: contract.Policy{Severity: "error", Action: "quarantine"}},
			{ID: "amount", Kind: "record", Path: "$.amount", Predicate: contract.Predicate{Op: "range", Min: &min, Max: &max}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
		},
	}
	violations := Check(c, Record{"email": "invalid", "amount": 101.0})
	if len(violations) != 2 {
		t.Fatalf("got %d violations, want 2: %+v", len(violations), violations)
	}
	if violations[0].Action != "quarantine" || violations[1].Action != "reject" {
		t.Fatalf("unexpected actions: %+v", violations)
	}
}

func TestCheckNestedRequiredAndInteger(t *testing.T) {
	c := contract.Contract{
		IRVersion: "1.0",
		Metadata:  contract.Metadata{ID: "events", Version: "1"},
		Input:     contract.Input{Kind: "record-stream"},
		Rules: []contract.Rule{
			{ID: "customer", Kind: "record", Path: "$.customer.id", Predicate: contract.Predicate{Op: "required"}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
			{ID: "count", Kind: "record", Path: "$.count", Predicate: contract.Predicate{Op: "type", Type: "integer"}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
		},
	}
	violations := Check(c, Record{"customer": map[string]any{}, "count": 1.5})
	if len(violations) != 2 {
		t.Fatalf("got %d violations, want 2: %+v", len(violations), violations)
	}
}

func BenchmarkCheck(b *testing.B) {
	c := contract.Contract{
		IRVersion: "1.0",
		Metadata:  contract.Metadata{ID: "bench", Version: "1"},
		Input:     contract.Input{Kind: "record-stream"},
		Rules: []contract.Rule{
			{ID: "id", Kind: "record", Path: "$.id", Predicate: contract.Predicate{Op: "required"}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
			{ID: "email", Kind: "record", Path: "$.email", Predicate: contract.Predicate{Op: "format", Format: "email"}, OnBreach: contract.Policy{Severity: "error", Action: "quarantine"}},
			{ID: "amount", Kind: "record", Path: "$.amount", Predicate: contract.Predicate{Op: "range", Min: floatPtr(0), Max: floatPtr(10000)}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
		},
	}
	record := Record{"id": "123", "email": "person@example.com", "amount": 42.0}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Check(c, record)
	}
}

func floatPtr(value float64) *float64 { return &value }
