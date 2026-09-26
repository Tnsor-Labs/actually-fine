// Package brokoli contains the Brokoli-facing execution boundary.
//
// It intentionally depends only on actually-fine's public contract, engine,
// and result packages. A Brokoli plugin can adapt its dataset and evidence
// APIs to these small interfaces without importing Brokoli types into the
// engine.
package brokoli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Tnsor-Labs/actually-fine/contract"
	"github.com/Tnsor-Labs/actually-fine/engine"
	"github.com/Tnsor-Labs/actually-fine/result"
)

type InvalidRecordPolicy string

const (
	InvalidRecordFail       InvalidRecordPolicy = "fail"
	InvalidRecordQuarantine InvalidRecordPolicy = "quarantine"
)

// RecordSink receives the original NDJSON record, preserving the dataset
// representation supplied by the host.
type RecordSink func([]byte) error

// EvidenceSink receives validated result protocol events.
type EvidenceSink func(result.Event) error

type Request struct {
	Contract      contract.Contract
	Input         io.Reader
	Accepted      RecordSink
	Quarantined   RecordSink
	Evidence      EvidenceSink
	InvalidRecord InvalidRecordPolicy
	Context       context.Context
}

type Summary struct {
	Total       int
	Cleared     int
	Warnings    int
	Quarantined int
	Breached    int
	Halted      bool
}

func (s Summary) ExitCode() result.ExitCode {
	if s.Quarantined > 0 || s.Breached > 0 {
		return result.Breach
	}
	return result.Cleared
}

// Run executes one gate over an NDJSON dataset. Contract validation happens
// before input processing, which lets a host reject bad configuration before
// starting a pipeline task.
func Run(request Request) (Summary, error) {
	if err := request.Contract.Validate(); err != nil {
		return Summary{}, fmt.Errorf("invalid contract: %w", err)
	}
	if request.Input == nil {
		return Summary{}, fmt.Errorf("input is required")
	}
	if request.InvalidRecord == "" {
		request.InvalidRecord = InvalidRecordFail
	}
	if request.InvalidRecord != InvalidRecordFail && request.InvalidRecord != InvalidRecordQuarantine {
		return Summary{}, fmt.Errorf("unsupported invalid record policy %q", request.InvalidRecord)
	}
	ctx := request.Context
	if ctx == nil {
		ctx = context.Background()
	}

	var summary Summary
	checker := engine.NewStreamChecker(request.Contract)
	scanner := bufio.NewScanner(request.Input)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	line := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		line++
		raw := append([]byte(nil), scanner.Bytes()...)
		summary.Total++
		var record engine.Record
		parseErr := json.Unmarshal(raw, &record)
		if parseErr == nil && record == nil {
			parseErr = fmt.Errorf("record must be a JSON object")
		}
		if parseErr != nil {
			if request.InvalidRecord == InvalidRecordFail {
				return summary, fmt.Errorf("invalid JSON at line %d: %w", line, parseErr)
			}
			summary.Quarantined++
			if request.Quarantined != nil {
				if err := request.Quarantined(raw); err != nil {
					return summary, fmt.Errorf("quarantine line %d: %w", line, err)
				}
			}
			event := result.Event{ResultVersion: result.Version, Status: "breach", ContractID: request.Contract.Metadata.ID, ContractVersion: request.Contract.Metadata.Version, RuleID: "input.parse", RuleVersion: "1", Line: line, Path: "$", Severity: "error", Action: "quarantine", Message: fmt.Sprintf("invalid JSON input: %v", parseErr)}
			if err := publish(request.Evidence, event); err != nil {
				return summary, err
			}
			continue
		}

		violations := checker.Check(record)
		hasBreach, quarantine, halt := false, false, false
		for _, violation := range violations {
			if violation.Action == "warn" {
				summary.Warnings++
			} else {
				hasBreach = true
			}
			quarantine = quarantine || violation.Action == "quarantine"
			halt = halt || violation.Action == "halt"
			event := makeEvent(request.Contract, record, line, violation)
			if err := publish(request.Evidence, event); err != nil {
				return summary, err
			}
		}
		switch {
		case !hasBreach:
			summary.Cleared++
			if request.Accepted != nil {
				if err := request.Accepted(raw); err != nil {
					return summary, fmt.Errorf("accept line %d: %w", line, err)
				}
			}
		case quarantine:
			summary.Quarantined++
			if request.Quarantined != nil {
				if err := request.Quarantined(raw); err != nil {
					return summary, fmt.Errorf("quarantine line %d: %w", line, err)
				}
			}
		default:
			summary.Breached++
		}
		if halt {
			summary.Halted = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return summary, fmt.Errorf("read input: %w", err)
	}
	if !summary.Halted {
		for _, violation := range checker.Finalize() {
			if violation.Action == "warn" {
				summary.Warnings++
			} else {
				summary.Breached++
			}
			if err := publish(request.Evidence, makeEvent(request.Contract, engine.Record{}, max(1, line), violation)); err != nil {
				return summary, err
			}
		}
	}
	return summary, nil
}

func publish(sink EvidenceSink, event result.Event) error {
	if err := event.Validate(); err != nil {
		return err
	}
	if sink == nil {
		return nil
	}
	return sink(event)
}

func makeEvent(c contract.Contract, record engine.Record, line int, violation engine.Violation) result.Event {
	status := "breach"
	if violation.Action == "warn" {
		status = "warning"
	}
	return result.Event{ResultVersion: result.Version, Status: status, ContractID: c.Metadata.ID, ContractVersion: c.Metadata.Version, RuleID: violation.RuleID, RuleVersion: violation.RuleVersion, Line: line, RecordID: record["id"], Path: violation.Path, Severity: violation.Severity, Action: violation.Action, Message: violation.Message, SuggestedFix: violation.SuggestedFix, Value: violation.Value}
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
