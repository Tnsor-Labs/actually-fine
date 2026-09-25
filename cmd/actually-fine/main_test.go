package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Tnsor-Labs/actually-fine/result"
)

func TestRunRoutesQuarantineAndAcceptedRecords(t *testing.T) {
	dir := t.TempDir()
	contractPath := filepath.Join(dir, "contract.json")
	inputPath := filepath.Join(dir, "input.ndjson")
	validPath := filepath.Join(dir, "valid.ndjson")
	quarantinePath := filepath.Join(dir, "quarantine.ndjson")
	resultsPath := filepath.Join(dir, "results.jsonl")

	contractData := `{"ir_version":"1.0","contract":{"id":"test","version":"1"},"input":{"kind":"record-stream"},"rules":[{"id":"email","kind":"record","path":"$.email","predicate":{"op":"format","format":"email"},"on_breach":{"severity":"error","action":"quarantine"}}]}`
	inputData := "{\"email\":\"ok@example.com\"}\n{\"email\":\"bad\"}\n"
	if err := os.WriteFile(contractPath, []byte(contractData), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, []byte(inputData), 0o600); err != nil {
		t.Fatal(err)
	}

	status := run([]string{
		"--contract", contractPath,
		"--input", inputPath,
		"--valid-output", validPath,
		"--quarantine-output", quarantinePath,
		"--results", resultsPath,
	})
	if status != int(result.Breach) {
		t.Fatalf("run() status = %d, want %d", status, result.Breach)
	}
	valid, err := os.ReadFile(validPath)
	if err != nil {
		t.Fatal(err)
	}
	quarantine, err := os.ReadFile(quarantinePath)
	if err != nil {
		t.Fatal(err)
	}
	results, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(valid) != "{\"email\":\"ok@example.com\"}\n" {
		t.Fatalf("unexpected valid output: %q", valid)
	}
	if string(quarantine) != "{\"email\":\"bad\"}\n" {
		t.Fatalf("unexpected quarantine output: %q", quarantine)
	}
	if len(results) == 0 {
		t.Fatal("expected a breach result")
	}
}
