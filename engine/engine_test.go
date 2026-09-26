package engine

import (
	"encoding/json"
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

// Records do not only come from encoding/json. A Go caller building a
// Record passes ints; Brokoli decodes whole numbers as int64 to keep 64-bit
// values exact; a SQL driver returns int64 for integer columns. numberValue
// accepted only float64, so every one of those failed range and the
// number/integer type checks -- reported as "outside the allowed range" or
// "wrong type", and quarantined or rejected, for values that were fine.
//
// Found through Brokoli: the same pipeline cleared 5 of 10 rows when its
// data was held in memory as float64 and 0 of 10 when the same data came
// back from a spilled blob as int64.
func TestNumericPredicatesAcceptEveryGoNumber(t *testing.T) {
	rules := []contract.Rule{
		{ID: "range", Kind: "record", Path: "$.n", Predicate: contract.Predicate{Op: "range", Min: floatPtr(0), Max: floatPtr(5)}, OnBreach: contract.Policy{Action: "reject"}},
		{ID: "number", Kind: "record", Path: "$.n", Predicate: contract.Predicate{Op: "type", Type: "number"}, OnBreach: contract.Policy{Action: "reject"}},
	}
	c := contract.Contract{IRVersion: "1.0", Metadata: contract.Metadata{ID: "n", Version: "1"},
		Input: contract.Input{Kind: "record-stream"}, Rules: rules}

	for name, value := range map[string]any{
		"float64": 3.0, "float32": float32(3), "int": 3, "int8": int8(3), "int16": int16(3),
		"int32": int32(3), "int64": int64(3), "uint": uint(3), "uint8": uint8(3), "uint16": uint16(3),
		"uint32": uint32(3), "uint64": uint64(3), "json.Number": json.Number("3"),
	} {
		if got := Check(c, Record{"n": value}); len(got) != 0 {
			t.Errorf("%s 3 is a number inside [0, 5] but got violations %+v", name, got)
		}
	}
}

// The control: accepting every Go number must not mean accepting every
// value. Out of range is still out of range whatever the type, and
// things that are not numbers are still not numbers.
func TestNumericPredicatesStillRefuseWhatTheyShould(t *testing.T) {
	rangeRule := contract.Contract{IRVersion: "1.0", Metadata: contract.Metadata{ID: "n", Version: "1"},
		Input: contract.Input{Kind: "record-stream"},
		Rules: []contract.Rule{{ID: "range", Kind: "record", Path: "$.n",
			Predicate: contract.Predicate{Op: "range", Max: floatPtr(5)}, OnBreach: contract.Policy{Action: "reject"}}}}
	for name, value := range map[string]any{"int64": int64(6), "uint": uint(6), "json.Number": json.Number("6")} {
		if got := Check(rangeRule, Record{"n": value}); len(got) != 1 {
			t.Errorf("%s 6 is above max 5 and must breach; got %+v", name, got)
		}
	}

	numberRule := contract.Contract{IRVersion: "1.0", Metadata: contract.Metadata{ID: "n", Version: "1"},
		Input: contract.Input{Kind: "record-stream"},
		Rules: []contract.Rule{{ID: "number", Kind: "record", Path: "$.n",
			Predicate: contract.Predicate{Op: "type", Type: "number"}, OnBreach: contract.Policy{Action: "reject"}}}}
	for name, value := range map[string]any{"string": "3", "bool": true, "bad json.Number": json.Number("three")} {
		if got := Check(numberRule, Record{"n": value}); len(got) != 1 {
			t.Errorf("%s is not a number and must breach type=number; got %+v", name, got)
		}
	}
}

// integer is decided on the value, not on its Go type's name: every
// integer kind is an integer, and a float is one only when it is whole.
// Checking an int64 by converting it to float64 first is also wrong past
// 2^53, where the conversion rounds, so integer kinds never go through it.
func TestIntegerTypeCheck(t *testing.T) {
	c := contract.Contract{IRVersion: "1.0", Metadata: contract.Metadata{ID: "n", Version: "1"},
		Input: contract.Input{Kind: "record-stream"},
		Rules: []contract.Rule{{ID: "int", Kind: "record", Path: "$.n",
			Predicate: contract.Predicate{Op: "type", Type: "integer"}, OnBreach: contract.Policy{Action: "reject"}}}}

	for name, value := range map[string]any{
		"int64": int64(7), "max int64": int64(9223372036854775807), "uint64": uint64(18446744073709551615),
		"whole float64": 7.0, "json.Number int": json.Number("7"),
	} {
		if got := Check(c, Record{"n": value}); len(got) != 0 {
			t.Errorf("%s is an integer; got %+v", name, got)
		}
	}
	for name, value := range map[string]any{
		"fractional float64": 7.5, "fractional float32": float32(7.5), "json.Number fraction": json.Number("7.5"), "string": "7",
	} {
		if got := Check(c, Record{"n": value}); len(got) != 1 {
			t.Errorf("%s is not an integer and must breach; got %+v", name, got)
		}
	}
}
