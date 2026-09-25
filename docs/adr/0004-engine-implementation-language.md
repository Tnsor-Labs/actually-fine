# ADR 0004: Engine Implementation Language

- Status: Proposed
- Date: 2026-09-25

## Context

The engine must be fast, embeddable, easy to distribute, and compatible with
the Tnsor Labs ecosystem. Go aligns with Brokoli's existing core and single
binary model. Rust offers strong memory safety, attractive embedded and WASM
options, and potentially useful sandboxing primitives.

The implementation language must not determine the contract IR or SDK API.

## Decision

No final language decision is made by this ADR. The IR-first architecture lets
the project evaluate Go and Rust against the same conformance suite.

The decision must be based on a small prototype that measures:

- Streaming throughput and per-record latency.
- Memory use for large and long-lived streams.
- CLI and single-binary distribution.
- Embedding from Brokoli.
- Extension and sandboxing options.
- Developer velocity and ecosystem fit.

## Consequences

- Language selection is deferred without blocking IR and SDK design.
- A prototype and benchmark suite are required before implementation begins
  in earnest.
- The first engine language must implement the same versioned IR conformance
  tests that future engines will use.
