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

func TestCanonicalDigestNormalizesDefaultsAndRuleOrder(t *testing.T) {
	first := []byte(`{"ir_version":"1.0","contract":{"id":"orders","version":"1"},"input":{"kind":"record-stream"},"rules":[{"id":"z-rule","kind":"record","path":"$.z","predicate":{"op":"required"},"on_breach":{"action":"reject"}},{"id":"a-rule","version":"1","kind":"record","path":"$.a","predicate":{"op":"required"},"on_breach":{"severity":"error","action":"reject"}}]}`)
	second := []byte(`{"rules":[{"on_breach":{"action":"reject","severity":"error"},"predicate":{"op":"required"},"path":"$.a","kind":"record","id":"a-rule","version":"1"},{"id":"z-rule","version":"1","kind":"record","path":"$.z","predicate":{"op":"required"},"on_breach":{"severity":"error","action":"reject"}}],"input":{"kind":"record-stream"},"contract":{"version":"1","id":"orders"},"ir_version":"1.0"}`)
	c1, err := Decode(first)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := Decode(second)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := CanonicalJSON(c1)
	if err != nil {
		t.Fatal(err)
	}
	wantCanonical := `{"ir_version":"1.0","contract":{"id":"orders","version":"1"},"input":{"kind":"record-stream"},"rules":[{"id":"a-rule","version":"1","kind":"record","path":"$.a","predicate":{"op":"required"},"on_breach":{"severity":"error","action":"reject"}},{"id":"z-rule","version":"1","kind":"record","path":"$.z","predicate":{"op":"required"},"on_breach":{"severity":"error","action":"reject"}}]}`
	if string(canonical) != wantCanonical {
		t.Fatalf("canonical JSON = %s, want %s", canonical, wantCanonical)
	}
	digest1, err := Digest(c1)
	if err != nil {
		t.Fatal(err)
	}
	digest2, err := Digest(c2)
	if err != nil {
		t.Fatal(err)
	}
	if digest1 != digest2 || len(digest1) != len("sha256:")+64 {
		t.Fatalf("digests differ or have wrong format: %q and %q", digest1, digest2)
	}
}

func TestDecodeRejectsUnknownAndTrailingData(t *testing.T) {
	base := `{"ir_version":"1.0","contract":{"id":"x","version":"1"},"input":{"kind":"record-stream"},"rules":[{"id":"r","kind":"record","path":"$.x","predicate":{"op":"required"},"on_breach":{"action":"reject"}}]}`
	if _, err := Decode([]byte(`{"extra":true,` + base[1:])); err == nil {
		t.Fatal("Decode() accepted an unknown field")
	}
	if _, err := Decode([]byte(base + base)); err == nil {
		t.Fatal("Decode() accepted trailing JSON")
	}
}
