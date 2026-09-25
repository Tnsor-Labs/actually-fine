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

func TestCheckDistinguishesMissingAndNull(t *testing.T) {
	c := contract.Contract{
		IRVersion: "1.0",
		Metadata:  contract.Metadata{ID: "events", Version: "1"},
		Input:     contract.Input{Kind: "record-stream"},
		Rules: []contract.Rule{
			{ID: "required", Kind: "record", Path: "$.required", Predicate: contract.Predicate{Op: "required"}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
			{ID: "not-null", Kind: "record", Path: "$.nullable", Predicate: contract.Predicate{Op: "not_null"}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
		},
	}
	violations := Check(c, Record{"nullable": nil})
	if len(violations) != 2 {
		t.Fatalf("got %d violations, want 2: %+v", len(violations), violations)
	}
	if violations[0].Message != "field is missing" || violations[1].Message != "field must not be null" {
		t.Fatalf("unexpected null/missing messages: %+v", violations)
	}
}

func TestStreamCheckerUniqueAndCount(t *testing.T) {
	c := contract.Contract{
		IRVersion: "1.0",
		Metadata:  contract.Metadata{ID: "events", Version: "1"},
		Input:     contract.Input{Kind: "record-stream"},
		Rules: []contract.Rule{
			{ID: "id-unique", Kind: "stream", Path: "$.id", Predicate: contract.Predicate{Op: "unique"}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
			{ID: "minimum-count", Kind: "stream", Path: "$", Predicate: contract.Predicate{Op: "count", Min: floatPtr(3)}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
		},
	}
	checker := NewStreamChecker(c)
	if got := checker.Check(Record{"id": "a"}); len(got) != 0 {
		t.Fatalf("first record violations = %+v", got)
	}
	if got := checker.Check(Record{"id": "a"}); len(got) != 1 || got[0].RuleID != "id-unique" {
		t.Fatalf("duplicate violations = %+v", got)
	}
	if got := checker.Finalize(); len(got) != 1 || got[0].RuleID != "minimum-count" {
		t.Fatalf("final violations = %+v", got)
	}
}

func TestEnumComparisonIsStrict(t *testing.T) {
	c := contract.Contract{
		IRVersion: "1.0",
		Metadata:  contract.Metadata{ID: "events", Version: "1"},
		Input:     contract.Input{Kind: "record-stream"},
		Rules: []contract.Rule{
			{ID: "status", Kind: "record", Path: "$.status", Predicate: contract.Predicate{Op: "enum", Values: []any{"1"}}, OnBreach: contract.Policy{Severity: "error", Action: "reject"}},
		},
	}
	if got := Check(c, Record{"status": 1.0}); len(got) != 1 {
		t.Fatal("numeric value incorrectly matched string enum")
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
