package arrow

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/Tnsor-Labs/actually-fine/engine"
	"github.com/Tnsor-Labs/actually-fine/transport"
)

func TestRoundTripPreservesLogicalValues(t *testing.T) {
	columns := []string{"id", "name", "score", "active", "missing"}
	rows := []engine.Record{{"id": int64(9007199254740993), "name": "first", "score": 1.5, "active": true}, {"id": int64(2), "name": "second", "score": 2.0, "active": false, "missing": nil}}
	var stream bytes.Buffer
	if err := Encode(&stream, columns, rows); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(&stream)
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.Rows[0]["id"]; got != int64(9007199254740993) {
		t.Fatalf("id lost precision or type: %#v (%T)", got, got)
	}
	if got := decoded.Rows[0]["score"]; got != 1.5 {
		t.Fatalf("unexpected score: %#v (%T)", got, got)
	}
	if decoded.Rows[1]["missing"] != nil {
		t.Fatalf("expected null value, got %#v", decoded.Rows[1]["missing"])
	}
}

func TestBatchReaderWriterAvoidsNDJSONConversion(t *testing.T) {
	var stream bytes.Buffer
	writer := NewBatchWriter(&stream)
	columns := []string{"id", "name"}
	if err := writer.Write(context.Background(), transport.RecordBatch{Columns: columns, Records: []map[string]any{{"id": int64(1), "name": "a"}}}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Write(context.Background(), transport.RecordBatch{Columns: columns, Records: []map[string]any{{"id": int64(2), "name": "b"}}}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := NewBatchReader(&stream)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	first, err := reader.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := reader.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Records[0]["id"] != int64(1) || second.Records[0]["id"] != int64(2) {
		t.Fatalf("unexpected batches: %#v %#v", first.Records, second.Records)
	}
	if _, err := reader.Next(context.Background()); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}
