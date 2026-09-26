package brokoli

import (
	"context"
	"io"
	"testing"

	"github.com/Tnsor-Labs/actually-fine/contract"
	"github.com/Tnsor-Labs/actually-fine/result"
	"github.com/Tnsor-Labs/actually-fine/transport"
)

type testBatchSource struct {
	batches []transport.RecordBatch
	index   int
}

func (s *testBatchSource) Next(context.Context) (transport.RecordBatch, error) {
	if s.index == len(s.batches) {
		return transport.RecordBatch{}, io.EOF
	}
	batch := s.batches[s.index]
	s.index++
	return batch, nil
}

func TestRunBatchesRoutesWithoutRecordSerialization(t *testing.T) {
	c := contract.Contract{IRVersion: "1.0", Metadata: contract.Metadata{ID: "orders", Version: "1"}, Input: contract.Input{Kind: "record-stream"}, Rules: []contract.Rule{{ID: "required-email", Kind: "record", Path: "$.email", Predicate: contract.Predicate{Op: "required"}, OnBreach: contract.Policy{Action: "quarantine"}}}}
	source := &testBatchSource{batches: []transport.RecordBatch{{Columns: []string{"id", "email"}, Records: []map[string]any{{"id": "ok", "email": "a@example.com"}, {"id": "bad"}}}}}
	var accepted, quarantined transport.RecordBatch
	events := 0
	summary, err := RunBatches(BatchRequest{Contract: c, Source: source, Accepted: func(_ context.Context, batch transport.RecordBatch) error { accepted = batch; return nil }, Quarantined: func(_ context.Context, batch transport.RecordBatch) error { quarantined = batch; return nil }, Evidence: func(event result.Event) error { events++; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Cleared != 1 || summary.Quarantined != 1 || summary.ExitCode() != result.Breach || events != 1 {
		t.Fatalf("unexpected result: summary=%+v events=%d", summary, events)
	}
	if accepted.Records[0]["id"] != "ok" || quarantined.Records[0]["id"] != "bad" {
		t.Fatalf("unexpected routing: accepted=%v quarantined=%v", accepted.Records, quarantined.Records)
	}
}
