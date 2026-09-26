package arrow

import (
	"bytes"
	"context"
	"testing"

	"github.com/Tnsor-Labs/actually-fine/contract"
	"github.com/Tnsor-Labs/actually-fine/result"
	"github.com/Tnsor-Labs/actually-fine/transport"
)

func TestRunGateRoutesArrowBatches(t *testing.T) {
	contractIR := contract.Contract{IRVersion: "1.0", Metadata: contract.Metadata{ID: "orders", Version: "1"}, Input: contract.Input{Kind: "record-stream"}, Rules: []contract.Rule{{ID: "required-email", Kind: "record", Path: "$.email", Predicate: contract.Predicate{Op: "required"}, OnBreach: contract.Policy{Action: "quarantine"}}}}
	var input, accepted, quarantined bytes.Buffer
	inputWriter := NewBatchWriter(&input)
	if err := inputWriter.Write(context.Background(), transport.RecordBatch{Columns: []string{"id", "email"}, Records: []map[string]any{{"id": int64(1), "email": "a@example.com"}, {"id": int64(2)}}}); err != nil {
		t.Fatal(err)
	}
	if err := inputWriter.Close(); err != nil {
		t.Fatal(err)
	}
	events := 0
	summary, err := RunGate(GateRequest{Contract: contractIR, Input: &input, Accepted: &accepted, Quarantined: &quarantined, Evidence: func(result.Event) error { events++; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Cleared != 1 || summary.Quarantined != 1 || events != 1 {
		t.Fatalf("unexpected summary: %+v, events=%d", summary, events)
	}
	acceptedData, err := Decode(&accepted)
	if err != nil {
		t.Fatal(err)
	}
	quarantinedData, err := Decode(&quarantined)
	if err != nil {
		t.Fatal(err)
	}
	if acceptedData.Rows[0]["id"] != int64(1) || quarantinedData.Rows[0]["id"] != int64(2) {
		t.Fatalf("unexpected output rows: accepted=%v quarantined=%v", acceptedData.Rows, quarantinedData.Rows)
	}
}

func BenchmarkRunGate(b *testing.B) {
	contractIR := contract.Contract{IRVersion: "1.0", Metadata: contract.Metadata{ID: "orders", Version: "1"}, Input: contract.Input{Kind: "record-stream"}, Rules: []contract.Rule{{ID: "required-email", Kind: "record", Path: "$.email", Predicate: contract.Predicate{Op: "required"}, OnBreach: contract.Policy{Action: "quarantine"}}}}
	var input bytes.Buffer
	inputWriter := NewBatchWriter(&input)
	rows := make([]map[string]any, 10000)
	for i := range rows {
		rows[i] = map[string]any{"id": int64(i), "email": "user@example.com"}
	}
	if err := inputWriter.Write(context.Background(), transport.RecordBatch{Columns: []string{"id", "email"}, Records: rows}); err != nil {
		b.Fatal(err)
	}
	if err := inputWriter.Close(); err != nil {
		b.Fatal(err)
	}
	data := append([]byte(nil), input.Bytes()...)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RunGate(GateRequest{Contract: contractIR, Input: bytes.NewReader(data)}); err != nil {
			b.Fatal(err)
		}
	}
}
