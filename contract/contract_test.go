package contract

import "testing"

func TestDecodeValidContract(t *testing.T) {
	data := []byte(`{
      "ir_version":"1.0",
      "contract":{"id":"orders","version":"1"},
      "input":{"kind":"record-stream"},
      "rules":[{"id":"id-required","kind":"record","path":"$.id","predicate":{"op":"required"},"on_breach":{"severity":"error","action":"reject"}}]
    }`)
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Metadata.ID != "orders" || len(got.Rules) != 1 {
		t.Fatalf("unexpected contract: %+v", got)
	}
}

func TestDecodeRejectsInvalidIRVersion(t *testing.T) {
	data := []byte(`{"ir_version":"2.0","contract":{"id":"x","version":"1"},"input":{"kind":"record-stream"},"rules":[{"id":"r","kind":"record","path":"$.x","predicate":{"op":"required"},"on_breach":{"severity":"error","action":"reject"}}]}`)
	if _, err := Decode(data); err == nil {
		t.Fatal("Decode() accepted unsupported IR version")
	}
}

func TestDecodeRejectsInvalidRegex(t *testing.T) {
	data := []byte(`{"ir_version":"1.0","contract":{"id":"x","version":"1"},"input":{"kind":"record-stream"},"rules":[{"id":"r","kind":"record","path":"$.x","predicate":{"op":"regex","pattern":"["},"on_breach":{"severity":"error","action":"reject"}}]}`)
	if _, err := Decode(data); err == nil {
		t.Fatal("Decode() accepted invalid regex")
	}
}
