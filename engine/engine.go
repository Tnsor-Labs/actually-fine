package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/mail"
	"regexp"
	"strings"

	"github.com/Tnsor-Labs/actually-fine/contract"
)

type Record map[string]any

type Violation struct {
	RuleID       string `json:"rule"`
	RuleVersion  string `json:"rule_version"`
	Path         string `json:"path"`
	Severity     string `json:"severity"`
	Action       string `json:"action"`
	Message      string `json:"message"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
	Value        any    `json:"value,omitempty"`
}

func Check(c contract.Contract, record Record) []Violation {
	violations := make([]Violation, 0)
	for _, rule := range c.Rules {
		if rule.Kind != "record" {
			continue
		}
		value, present := lookup(record, rule.Path)
		message, failed := evaluate(rule, value, present)
		if failed {
			severity := rule.OnBreach.Severity
			if severity == "" {
				severity = "error"
			}
			violations = append(violations, violation(rule, message, value, severity))
		}
	}
	return violations
}

func lookup(record Record, path string) (any, bool) {
	if path == "$" {
		return record, true
	}
	parts := strings.Split(strings.TrimPrefix(path, "$."), ".")
	var current any = map[string]any(record)
	for _, part := range parts {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func evaluate(rule contract.Rule, value any, present bool) (string, bool) {
	p := rule.Predicate
	switch p.Op {
	case "required":
		if !present {
			return "field is missing", true
		}
		if value == nil {
			return "field must not be null", true
		}
		if value == "" {
			return "field must not be empty", true
		}
	case "not_null":
		if !present {
			return "field is missing", true
		}
		if value == nil {
			return "field must not be null", true
		}
	case "type":
		if !present || value == nil || !typeMatches(value, p.Type) {
			return fmt.Sprintf("value must have type %q", p.Type), true
		}
	case "format":
		if !present || value == nil || p.Format == "" || !formatMatches(value, p.Format) {
			return fmt.Sprintf("value must match format %q", p.Format), true
		}
	case "regex":
		text, ok := value.(string)
		matched := false
		if ok {
			matched, _ = regexp.MatchString(p.Pattern, text)
		}
		if !matched {
			return "value does not match pattern", true
		}
	case "range":
		number, ok := numberValue(value)
		if !ok || (p.Min != nil && number < *p.Min) || (p.Max != nil && number > *p.Max) {
			return "value is outside the allowed range", true
		}
	case "enum":
		matched := false
		for _, allowed := range p.Values {
			if equalJSON(allowed, value) {
				matched = true
				break
			}
		}
		if !matched {
			return "value is not an allowed member", true
		}
	default:
		return fmt.Sprintf("unsupported predicate %q", p.Op), true
	}
	return "", false
}

func violation(rule contract.Rule, message string, value any, severity string) Violation {
	if rule.Version == "" {
		rule.Version = "1"
	}
	return Violation{
		RuleID: rule.ID, RuleVersion: rule.Version, Path: rule.Path,
		Severity: severity, Action: rule.OnBreach.Action, Message: message,
		Value: value,
	}
}

func equalJSON(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

// StreamChecker evaluates record rules immediately and maintains only the
// state required by stream rules for the current execution.
type StreamChecker struct {
	contract contract.Contract
	seen     map[string]map[string]struct{}
	count    int
}

func NewStreamChecker(c contract.Contract) *StreamChecker {
	return &StreamChecker{contract: c, seen: make(map[string]map[string]struct{})}
}

func (s *StreamChecker) Check(record Record) []Violation {
	s.count++
	violations := Check(s.contract, record)
	for _, rule := range s.contract.Rules {
		if rule.Kind != "stream" || rule.Predicate.Op != "unique" {
			continue
		}
		value, present := lookup(record, rule.Path)
		if !present {
			violations = append(violations, violation(rule, "field is missing and cannot be unique", value, severity(rule)))
			continue
		}
		if value == nil {
			violations = append(violations, violation(rule, "field is null and cannot be unique", value, severity(rule)))
			continue
		}
		keyBytes, err := json.Marshal(value)
		if err != nil {
			violations = append(violations, violation(rule, "value cannot be compared for uniqueness", value, severity(rule)))
			continue
		}
		if s.seen[rule.ID] == nil {
			s.seen[rule.ID] = make(map[string]struct{})
		}
		key := string(keyBytes)
		if _, exists := s.seen[rule.ID][key]; exists {
			violations = append(violations, violation(rule, "value was already seen in this stream", value, severity(rule)))
			continue
		}
		s.seen[rule.ID][key] = struct{}{}
	}
	return violations
}

func (s *StreamChecker) Finalize() []Violation {
	violations := make([]Violation, 0)
	for _, rule := range s.contract.Rules {
		if rule.Kind != "stream" || rule.Predicate.Op != "count" {
			continue
		}
		count := float64(s.count)
		if (rule.Predicate.Min != nil && count < *rule.Predicate.Min) || (rule.Predicate.Max != nil && count > *rule.Predicate.Max) {
			violations = append(violations, violation(rule, fmt.Sprintf("stream count %d is outside the allowed range", s.count), s.count, severity(rule)))
		}
	}
	return violations
}

func severity(rule contract.Rule) string {
	if rule.OnBreach.Severity == "" {
		return "error"
	}
	return rule.OnBreach.Severity
}

func typeMatches(value any, expected string) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := numberValue(value)
		return ok
	case "integer":
		return isInteger(value)
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		text := fmt.Sprintf("%T", value)
		return strings.HasPrefix(text, "[]")
	default:
		return false
	}
}

// numberValue reads a record value as a float64 when it is a number of
// any Go numeric type, or a json.Number that parses.
//
// Records do not only come from encoding/json, which is the one source
// that yields float64 for every number. A Go caller builds a Record with
// ints; a SQL driver returns int64 for integer columns; a caller that
// keeps 64-bit values exact decodes whole numbers as int64. Accepting only
// float64 made every one of those fail range and number/integer checks,
// reported as a range or type breach for a value that was fine.
//
// Range compares in float64, so an integer beyond 2^53 is compared at
// float64 precision. The bounds are float64 already, so this adds no
// imprecision the contract did not have.
func numberValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	}
	return 0, false
}

// isInteger reports whether a value is an integer.
//
// Integer kinds are integers by type and never go through float64: an
// int64 past 2^53 rounds when converted, and the old check compared a
// value with its own round trip through int64, which is undefined for a
// float outside int64's range. A float is an integer only when it is
// whole and finite.
func isInteger(value any) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return isWhole(v)
	case float32:
		return isWhole(float64(v))
	case json.Number:
		if _, err := v.Int64(); err == nil {
			return true
		}
		f, err := v.Float64()
		return err == nil && isWhole(f)
	}
	return false
}

func isWhole(f float64) bool {
	return !math.IsInf(f, 0) && !math.IsNaN(f) && f == math.Trunc(f)
}

func formatMatches(value any, format string) bool {
	text, ok := value.(string)
	if !ok {
		return false
	}
	switch format {
	case "email":
		address, err := mail.ParseAddress(text)
		return err == nil && address.Address == text
	default:
		return false
	}
}
