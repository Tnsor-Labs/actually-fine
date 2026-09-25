package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/Tnsor-Labs/actually-fine/contract"
	"github.com/Tnsor-Labs/actually-fine/engine"
	"github.com/Tnsor-Labs/actually-fine/result"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(int(result.InvalidContract))
	}
	var status int
	switch os.Args[1] {
	case "run":
		status = run(os.Args[2:])
	case "validate":
		status = validateContract(os.Args[2:])
	case "inspect":
		status = inspectContract(os.Args[2:])
	default:
		printUsage()
		status = int(result.InvalidContract)
	}
	os.Exit(status)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: actually-fine <run|validate|inspect> --contract contract.json [options]")
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
	invalidAction := flags.String("invalid-record", "fail", "invalid JSON policy: fail or quarantine")
	if err := flags.Parse(args); err != nil {
		return int(result.InvalidContract)
	}
	if *contractPath == "" {
		fmt.Fprintln(os.Stderr, "--contract is required")
		return int(result.InvalidContract)
	}
	if *format != "human" && *format != "jsonl" {
		fmt.Fprintln(os.Stderr, "--format must be human or jsonl")
		return int(result.InvalidContract)
	}
	if *invalidAction != "fail" && *invalidAction != "quarantine" {
		fmt.Fprintln(os.Stderr, "--invalid-record must be fail or quarantine")
		return int(result.InvalidContract)
	}

	c, err := readContract(*contractPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid contract: %v\n", err)
		return int(result.InvalidContract)
	}

	in, closeInput, err := openInput(*inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		return int(result.RuntimeFailure)
	}
	defer closeInput()
	valid, closeValid, err := createOutput(*validPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open valid output: %v\n", err)
		return int(result.RuntimeFailure)
	}
	defer closeValid()
	quarantine, closeQuarantine, err := createOutput(*quarantinePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open quarantine output: %v\n", err)
		return int(result.RuntimeFailure)
	}
	defer closeQuarantine()
	results, closeResults, err := createOutput(*resultsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open results output: %v\n", err)
		return int(result.RuntimeFailure)
	}
	defer closeResults()

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		if closer, ok := in.(io.Closer); ok {
			_ = closer.Close()
		}
	}()
	lineNumber, total, cleared, quarantined, breached, warnings := 0, 0, 0, 0, 0, 0
	interrupted, halted := false, false
	for scanner.Scan() {
		if ctx.Err() != nil {
			interrupted = true
			break
		}
		lineNumber++
		line := scanner.Bytes()
		total++
		var record engine.Record
		parseErr := json.Unmarshal(line, &record)
		if parseErr == nil && record == nil {
			parseErr = fmt.Errorf("record must be a JSON object")
		}
		if parseErr != nil {
			if *invalidAction == "quarantine" {
				quarantined++
				if quarantine != nil {
					if err := writeLine(quarantine, line); err != nil {
						fmt.Fprintf(os.Stderr, "write quarantine output: %v\n", err)
						return int(result.RuntimeFailure)
					}
				}
				e := result.Event{
					ResultVersion: result.Version, Status: "breach", ContractID: c.Metadata.ID,
					ContractVersion: c.Metadata.Version, RuleID: "input.parse", RuleVersion: "1",
					Line: lineNumber, Path: "$", Severity: "error", Action: "quarantine",
					Message: fmt.Sprintf("invalid JSON input: %v", parseErr),
				}
				if results != nil {
					if err := writeEvent(results, e); err != nil {
						fmt.Fprintf(os.Stderr, "write result: %v\n", err)
						return int(result.RuntimeFailure)
					}
				}
				if *format == "jsonl" {
					if err := writeEvent(os.Stdout, e); err != nil {
						return int(result.RuntimeFailure)
					}
				}
				continue
			}
			fmt.Fprintf(os.Stderr, "invalid JSON at line %d: %v\n", lineNumber, parseErr)
			return int(result.RuntimeFailure)
		}
		violations := engine.Check(c, record)
		hasBreach, shouldQuarantine, shouldHalt := false, false, false
		for _, violation := range violations {
			if violation.Action == "warn" {
				warnings++
			}
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
				e := makeEvent(c, record, lineNumber, violation)
				if err := writeEvent(results, e); err != nil {
					fmt.Fprintf(os.Stderr, "write result: %v\n", err)
					return int(result.RuntimeFailure)
				}
			}
			if *format == "jsonl" {
				e := makeEvent(c, record, lineNumber, violation)
				if err := writeEvent(os.Stdout, e); err != nil {
					return int(result.RuntimeFailure)
				}
			}
		}
		if !hasBreach {
			cleared++
			if valid != nil {
				if err := writeLine(valid, line); err != nil {
					fmt.Fprintf(os.Stderr, "write valid output: %v\n", err)
					return int(result.RuntimeFailure)
				}
			}
		} else if shouldQuarantine {
			quarantined++
			if quarantine != nil {
				if err := writeLine(quarantine, line); err != nil {
					fmt.Fprintf(os.Stderr, "write quarantine output: %v\n", err)
					return int(result.RuntimeFailure)
				}
			}
		} else {
			breached++
		}
		if shouldHalt {
			halted = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		return int(result.RuntimeFailure)
	}
	if *format == "human" {
		fmt.Fprintf(os.Stdout, "contract: %s\nrecords: %d\ncleared: %d\nwarnings: %d\nquarantined: %d\nbreached: %d\nhalted: %t\ninterrupted: %t\n", c.Metadata.ID, total, cleared, warnings, quarantined, breached, halted, interrupted)
	}
	if interrupted {
		return int(result.RuntimeFailure)
	}
	if quarantined > 0 || breached > 0 {
		return int(result.Breach)
	}
	return int(result.Cleared)
}

func validateContract(args []string) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	contractPath := flags.String("contract", "", "contract IR JSON file")
	if err := flags.Parse(args); err != nil {
		return int(result.InvalidContract)
	}
	if *contractPath == "" {
		fmt.Fprintln(os.Stderr, "--contract is required")
		return int(result.InvalidContract)
	}
	c, err := readContract(*contractPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid contract: %v\n", err)
		return int(result.InvalidContract)
	}
	digest, err := contract.Digest(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "digest contract: %v\n", err)
		return int(result.InvalidContract)
	}
	fmt.Fprintf(os.Stdout, "valid contract: %s@%s\ndigest: %s\n", c.Metadata.ID, c.Metadata.Version, digest)
	return int(result.Cleared)
}

func inspectContract(args []string) int {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	contractPath := flags.String("contract", "", "contract IR JSON file")
	if err := flags.Parse(args); err != nil {
		return int(result.InvalidContract)
	}
	if *contractPath == "" {
		fmt.Fprintln(os.Stderr, "--contract is required")
		return int(result.InvalidContract)
	}
	c, err := readContract(*contractPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid contract: %v\n", err)
		return int(result.InvalidContract)
	}
	canonical, err := contract.CanonicalJSON(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "canonicalize contract: %v\n", err)
		return int(result.InvalidContract)
	}
	digest, err := contract.Digest(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "digest contract: %v\n", err)
		return int(result.InvalidContract)
	}
	fmt.Fprintf(os.Stdout, "digest: %s\n%s\n", digest, canonical)
	return int(result.Cleared)
}

func readContract(path string) (contract.Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return contract.Contract{}, fmt.Errorf("read contract: %w", err)
	}
	return contract.Decode(data)
}

func makeEvent(c contract.Contract, record engine.Record, line int, violation engine.Violation) result.Event {
	status := "breach"
	if violation.Action == "warn" {
		status = "warning"
	}
	return result.Event{
		ResultVersion: result.Version, Status: status, ContractID: c.Metadata.ID,
		ContractVersion: c.Metadata.Version, RuleID: violation.RuleID,
		RuleVersion: violation.RuleVersion, Line: line, RecordID: record["id"],
		Path: violation.Path, Severity: violation.Severity, Action: violation.Action,
		Message: violation.Message, SuggestedFix: violation.SuggestedFix,
		Value: violation.Value,
	}
}

func writeEvent(writer io.Writer, event result.Event) error {
	if err := event.Validate(); err != nil {
		return err
	}
	return json.NewEncoder(writer).Encode(event)
}

func writeLine(writer io.Writer, line []byte) error {
	if _, err := writer.Write(line); err != nil {
		return err
	}
	_, err := writer.Write([]byte{'\n'})
	return err
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
