package contract

import (
	"encoding/json"
	"fmt"
	"regexp"
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
		if r.Kind != "record" {
			return fmt.Errorf("rule %q: kind must be %q", r.ID, "record")
		}
		if r.Path == "" || (r.Path != "$" && !strings.HasPrefix(r.Path, "$.")) {
			return fmt.Errorf("rule %q: path must be $ or start with $.", r.ID)
		}
		if r.Predicate.Op == "" {
			return fmt.Errorf("rule %q: predicate.op is required", r.ID)
		}
		if r.OnBreach.Severity == "" {
			r.OnBreach.Severity = "error"
		}
		if r.OnBreach.Severity != "warning" && r.OnBreach.Severity != "error" {
			return fmt.Errorf("rule %q: severity must be warning or error", r.ID)
		}
		if r.OnBreach.Action == "" {
			return fmt.Errorf("rule %q: action is required", r.ID)
		}
		if r.OnBreach.Action != "warn" && r.OnBreach.Action != "reject" && r.OnBreach.Action != "quarantine" && r.OnBreach.Action != "halt" {
			return fmt.Errorf("rule %q: unsupported action %q", r.ID, r.OnBreach.Action)
		}
		if r.Predicate.Op == "regex" && r.Predicate.Pattern == "" {
			return fmt.Errorf("rule %q: regex pattern is required", r.ID)
		}
		if r.Predicate.Op == "regex" {
			if _, err := regexp.Compile(r.Predicate.Pattern); err != nil {
				return fmt.Errorf("rule %q: invalid regex: %w", r.ID, err)
			}
		}
	}
	return nil
}

func Decode(data []byte) (Contract, error) {
	var c Contract
	if err := json.Unmarshal(data, &c); err != nil {
		return Contract{}, fmt.Errorf("decode contract: %w", err)
	}
	if err := c.Validate(); err != nil {
		return Contract{}, err
	}
	return c, nil
}
