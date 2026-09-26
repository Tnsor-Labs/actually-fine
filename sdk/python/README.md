# actually-fine Python SDK

The Python SDK is an authoring and IR compiler surface. It does not implement
rule execution separately from the Go engine.

```python
from actually_fine import Contract, Predicate, Rule

contract = Contract(
    "orders",
    "1",
    [Rule("email-valid", "$.email", Predicate("format", format="email"), "quarantine")],
)

contract.canonical_json()  # bytes suitable for the CLI or an adapter
contract.digest()          # sha256:<canonical JSON digest>
```

The compiler derives rule kind from the predicate, applies the same defaults as
the Go contract normalizer, sorts rules by ID, and rejects invalid IR before it
is emitted.
