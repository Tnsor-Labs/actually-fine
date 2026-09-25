package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Tnsor-Labs/actually-fine/contract"
	"github.com/Tnsor-Labs/actually-fine/engine"
)

const (
	exitCleared = 0
	exitBreach  = 1
	exitInvalid = 2
	exitRuntime = 3
	exitPolicy  = 4
)

type event struct {
	Status   string `json:"status"`
	Line     int    `json:"line"`
	Contract string `json:"contract"`
	Rule     string `json:"rule,omitempty"`
	Path     string `json:"path,omitempty"`
	Severity string `json:"severity,omitempty"`
	Action   string `json:"action,omitempty"`
	Message  string `json:"message,omitempty"`
	Value    any    `json:"value,omitempty"`
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintln(os.Stderr, "usage: actually-fine run --contract contract.json [--input file] [options]")
		os.Exit(exitInvalid)
	}
	os.Exit(run(os.Args[2:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	contractPath := flags.String("contract", "", "contract IR JSON file")
	inputPath := flags.String("input", "-", "NDJSON input file, or - for stdin")
	validPath := flags.String("valid-output", "", "accepted NDJSON output file")
	quarantinePath := flags.String("quarantine-output", "", "quarantined NDJSON output file")
	resultsPath := flags.String("results", "", "JSONL breach event output file")
	format := flags.String("format", "human", "human or jsonl")
	if err := flags.Parse(args); err != nil {
		return exitInvalid
	}
	if *contractPath == "" {
		fmt.Fprintln(os.Stderr, "--contract is required")
		return exitInvalid
	}
	if *format != "human" && *format != "jsonl" {
		fmt.Fprintln(os.Stderr, "--format must be human or jsonl")
		return exitInvalid
	}

	contractData, err := os.ReadFile(*contractPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read contract: %v\n", err)
		return exitInvalid
	}
	c, err := contract.Decode(contractData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid contract: %v\n", err)
		return exitInvalid
	}

	in, closeInput, err := openInput(*inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		return exitRuntime
	}
	defer closeInput()
	valid, closeValid, err := createOutput(*validPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open valid output: %v\n", err)
		return exitRuntime
	}
	defer closeValid()
	quarantine, closeQuarantine, err := createOutput(*quarantinePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open quarantine output: %v\n", err)
		return exitRuntime
	}
	defer closeQuarantine()
	results, closeResults, err := createOutput(*resultsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open results output: %v\n", err)
		return exitRuntime
	}
	defer closeResults()

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	lineNumber, total, cleared, quarantined, breached := 0, 0, 0, 0, 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		var record engine.Record
		if err := json.Unmarshal(line, &record); err != nil {
			fmt.Fprintf(os.Stderr, "invalid JSON at line %d: %v\n", lineNumber, err)
			return exitRuntime
		}
		total++
		violations := engine.Check(c, record)
		hasBreach, shouldQuarantine, shouldHalt := false, false, false
		for _, violation := range violations {
			if violation.Action != "warn" {
				hasBreach = true
			}
			if violation.Action == "quarantine" {
				shouldQuarantine = true
			}
			if violation.Action == "halt" {
				shouldHalt = true
			}
			if results != nil {
				status := "breach"
				if violation.Action == "warn" {
					status = "warning"
				}
				e := event{Status: status, Line: lineNumber, Contract: c.Metadata.ID, Rule: violation.RuleID, Path: violation.Path, Severity: violation.Severity, Action: violation.Action, Message: violation.Message, Value: violation.Value}
				if err := json.NewEncoder(results).Encode(e); err != nil {
					fmt.Fprintf(os.Stderr, "write result: %v\n", err)
					return exitRuntime
				}
			}
			if *format == "jsonl" {
				status := "breach"
				if violation.Action == "warn" {
					status = "warning"
				}
				e := event{Status: status, Line: lineNumber, Contract: c.Metadata.ID, Rule: violation.RuleID, Path: violation.Path, Severity: violation.Severity, Action: violation.Action, Message: violation.Message, Value: violation.Value}
				if err := json.NewEncoder(os.Stdout).Encode(e); err != nil {
					return exitRuntime
				}
			}
		}
		if !hasBreach {
			cleared++
			if valid != nil {
				if _, err := valid.Write(append(line, '\n')); err != nil {
					fmt.Fprintf(os.Stderr, "write valid output: %v\n", err)
					return exitRuntime
				}
			}
		} else if shouldQuarantine {
			quarantined++
			if quarantine != nil {
				if _, err := quarantine.Write(append(line, '\n')); err != nil {
					fmt.Fprintf(os.Stderr, "write quarantine output: %v\n", err)
					return exitRuntime
				}
			}
		} else {
			breached++
		}
		if shouldHalt {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		return exitRuntime
	}
	if *format == "human" {
		fmt.Fprintf(os.Stdout, "contract: %s\nrecords: %d\ncleared: %d\nquarantined: %d\nbreached: %d\n", c.Metadata.ID, total, cleared, quarantined, breached)
	}
	if quarantined > 0 || breached > 0 {
		return exitBreach
	}
	return exitCleared
}

func openInput(path string) (io.Reader, func(), error) {
	if path == "-" {
		return os.Stdin, func() {}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { _ = f.Close() }, nil
}

func createOutput(path string) (*os.File, func(), error) {
	if path == "" {
		return nil, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { _ = f.Close() }, nil
}
