package arrow

import (
	"bytes"
	"testing"

	"github.com/Tnsor-Labs/actually-fine/engine"
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
