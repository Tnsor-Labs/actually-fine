package contract

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

const CurrentIRVersion = "1.0"

type Contract struct {
	IRVersion string   `json:"ir_version"`
	Metadata  Metadata `json:"contract"`
	Input     Input    `json:"input"`
	Rules     []Rule   `json:"rules"`
}

type Metadata struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Input struct {
	Kind string `json:"kind"`
}

type Rule struct {
	ID        string    `json:"id"`
	Version   string    `json:"version,omitempty"`
	Kind      string    `json:"kind"`
	Path      string    `json:"path"`
	Predicate Predicate `json:"predicate"`
	OnBreach  Policy    `json:"on_breach"`
}

type Predicate struct {
	Op      string   `json:"op"`
	Type    string   `json:"type,omitempty"`
	Format  string   `json:"format,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Values  []any    `json:"values,omitempty"`
}

type Policy struct {
	Severity string `json:"severity"`
	Action   string `json:"action"`
}

func (c Contract) Validate() error {
	if c.IRVersion != CurrentIRVersion {
		return fmt.Errorf("unsupported ir_version %q, expected %q", c.IRVersion, CurrentIRVersion)
	}
	if c.Metadata.ID == "" || c.Metadata.Version == "" {
		return fmt.Errorf("contract id and version are required")
	}
	if c.Input.Kind != "record-stream" {
		return fmt.Errorf("input.kind must be %q", "record-stream")
	}
	if len(c.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}
	seen := make(map[string]bool, len(c.Rules))
	for _, r := range c.Rules {
		if r.ID == "" || seen[r.ID] {
			return fmt.Errorf("rule ids must be present and unique: %q", r.ID)
		}
		seen[r.ID] = true
		if r.Version != "" && r.Version != "1" {
			return fmt.Errorf("rule %q: unsupported version %q", r.ID, r.Version)
		}
		if r.Kind != "record" {
			return fmt.Errorf("rule %q: kind must be %q", r.ID, "record")
		}
		if r.Path == "" || (r.Path != "$" && !strings.HasPrefix(r.Path, "$.")) {
			return fmt.Errorf("rule %q: path must be $ or start with $.", r.ID)
		}
		if r.Predicate.Op == "" {
			return fmt.Errorf("rule %q: predicate.op is required", r.ID)
		}
		switch r.Predicate.Op {
		case "required":
		case "type":
			if r.Predicate.Type != "string" && r.Predicate.Type != "number" && r.Predicate.Type != "integer" && r.Predicate.Type != "boolean" && r.Predicate.Type != "object" && r.Predicate.Type != "array" {
				return fmt.Errorf("rule %q: unsupported type %q", r.ID, r.Predicate.Type)
			}
		case "format":
			if r.Predicate.Format != "email" {
				return fmt.Errorf("rule %q: unsupported format %q", r.ID, r.Predicate.Format)
			}
		case "regex":
			if r.Predicate.Pattern == "" {
				return fmt.Errorf("rule %q: regex pattern is required", r.ID)
			}
			if _, err := regexp.Compile(r.Predicate.Pattern); err != nil {
				return fmt.Errorf("rule %q: invalid regex: %w", r.ID, err)
			}
		case "range":
			if r.Predicate.Min == nil && r.Predicate.Max == nil {
				return fmt.Errorf("rule %q: range requires min or max", r.ID)
			}
			if r.Predicate.Min != nil && r.Predicate.Max != nil && *r.Predicate.Min > *r.Predicate.Max {
				return fmt.Errorf("rule %q: range min cannot exceed max", r.ID)
			}
		case "enum":
			if len(r.Predicate.Values) == 0 {
				return fmt.Errorf("rule %q: enum values are required", r.ID)
			}
		default:
			return fmt.Errorf("rule %q: unsupported predicate %q", r.ID, r.Predicate.Op)
		}
		if r.OnBreach.Severity != "" && r.OnBreach.Severity != "warning" && r.OnBreach.Severity != "error" {
			return fmt.Errorf("rule %q: severity must be warning or error", r.ID)
		}
		if r.OnBreach.Action == "" {
			return fmt.Errorf("rule %q: action is required", r.ID)
		}
		if r.OnBreach.Action != "warn" && r.OnBreach.Action != "reject" && r.OnBreach.Action != "quarantine" && r.OnBreach.Action != "halt" {
			return fmt.Errorf("rule %q: unsupported action %q", r.ID, r.OnBreach.Action)
		}
	}
	return nil
}

func Decode(data []byte) (Contract, error) {
	var c Contract
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return Contract{}, fmt.Errorf("decode contract: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Contract{}, fmt.Errorf("decode contract: multiple JSON values")
		}
		return Contract{}, fmt.Errorf("decode contract: trailing data: %w", err)
	}
	return c.Normalize()
}

// Normalize applies semantic defaults and sorts rules by identity. It returns
// a copy so callers can safely use the input contract for other purposes.
func (c Contract) Normalize() (Contract, error) {
	if err := c.Validate(); err != nil {
		return Contract{}, err
	}
	normalized := c
	normalized.Rules = append([]Rule(nil), c.Rules...)
	for i := range normalized.Rules {
		if normalized.Rules[i].Version == "" {
			normalized.Rules[i].Version = "1"
		}
		if normalized.Rules[i].OnBreach.Severity == "" {
			normalized.Rules[i].OnBreach.Severity = "error"
		}
	}
	sort.Slice(normalized.Rules, func(i, j int) bool {
		return normalized.Rules[i].ID < normalized.Rules[j].ID
	})
	return normalized, nil
}

func CanonicalJSON(c Contract) ([]byte, error) {
	normalized, err := c.Normalize()
	if err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

func Digest(c Contract) (string, error) {
	canonical, err := CanonicalJSON(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + fmt.Sprintf("%x", sum), nil
}
