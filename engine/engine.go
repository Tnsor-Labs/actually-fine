package engine

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/Tnsor-Labs/actually-fine/contract"
)

type Record map[string]any

type Violation struct {
	RuleID   string `json:"rule"`
	Path     string `json:"path"`
	Severity string `json:"severity"`
	Action   string `json:"action"`
	Message  string `json:"message"`
	Value    any    `json:"value,omitempty"`
}

func Check(c contract.Contract, record Record) []Violation {
	violations := make([]Violation, 0)
	for _, rule := range c.Rules {
		value, present := lookup(record, rule.Path)
		message, failed := evaluate(rule, value, present)
		if failed {
			severity := rule.OnBreach.Severity
			if severity == "" {
				severity = "error"
			}
			violations = append(violations, Violation{
				RuleID: rule.ID, Path: rule.Path, Severity: severity,
				Action: rule.OnBreach.Action, Message: message, Value: value,
			})
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
		if !present || value == nil || (value == "") {
			return "field is required", true
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
			if fmt.Sprint(allowed) == fmt.Sprint(value) {
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

func typeMatches(value any, expected string) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := numberValue(value)
		return ok
	case "integer":
		number, ok := numberValue(value)
		return ok && number == float64(int64(number))
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

func numberValue(value any) (float64, bool) {
	number, ok := value.(float64)
	return number, ok
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
