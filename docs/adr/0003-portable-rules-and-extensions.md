# ADR 0003: Portable Rules and Runtime Extensions

- Status: Proposed
- Date: 2026-09-25

## Context

An IR only improves interoperability if its semantics are executable outside
the language in which a contract was authored. Arbitrary Python, Go, or Rust
closures cannot be assumed to run in every adapter or engine.

At the same time, limiting users to a closed rule catalog would make the
engine impractical for domain-specific validation.

## Decision

The IR will distinguish between two rule classes:

1. Portable declarative rules with engine-defined semantics.
2. Explicit runtime extensions with a declared execution target.

Portable rules will be represented as structured IR, not as opaque strings.
Examples include nullability, type checks, ranges, formats, uniqueness,
freshness, reference existence, and stream-window constraints.

Runtime extensions will identify their runtime, module or artifact identity,
entrypoint, and required capabilities. An extension that the selected runtime
cannot execute must fail validation before processing data.

Arbitrary language closures will not be silently serialized as if they were
portable rules.

## Consequences

Positive:

- Equivalent SDKs have unambiguous semantics.
- The engine can optimize and inspect portable rules.
- Custom behavior remains possible without weakening compatibility guarantees.

Negative:

- The portable rule vocabulary requires careful design.
- Extensions need a separate packaging and sandboxing model.
- Some SDK conveniences will not be representable in portable IR.

## Open questions

- Whether the first extension target should be WASM, a process protocol, or a
  language-specific adapter.
- Which rule kinds require state and how that state is declared.
- Whether portable predicates should be modeled as an expression tree or a
  typed operation catalog.
