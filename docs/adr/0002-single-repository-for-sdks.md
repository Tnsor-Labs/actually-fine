# ADR 0002: Single Repository for SDKs

- Status: Accepted
- Date: 2026-09-25

## Context

actually-fine may eventually need SDKs for several languages used by the data
engineering ecosystem. Separate repositories would increase coordination cost
while the IR and engine behavior are still evolving.

The SDKs must be tested against the same fixtures and must remain behaviorally
identical when they compile equivalent contracts.

## Decision

The engine, CLI, IR schema, conformance fixtures, and future language SDKs will
live in one repository:

```text
github.com/Tnsor-Labs/actually-fine
```

The repository will use clear package boundaries and independent release
artifacts where the target ecosystem requires them. Repository co-location
does not make the SDKs runtime dependencies of the engine.

Development will proceed in this order:

1. Define and validate the IR.
2. Build and test the engine against the IR.
3. Prove the CLI and core product workflow end to end.
4. Add SDKs only after the engine provides a useful, stable product to author
   contracts for.

When SDK development begins, all SDKs will share:

- The canonical IR schema.
- Normalization and digest fixtures.
- Contract validation fixtures.
- Cross-language differential tests.

Splitting an SDK into a separate repository remains possible later if its
release cadence, community, or toolchain makes that beneficial. Such a split
must preserve the shared IR conformance suite.

## Consequences

Positive:

- IR changes and SDK updates can be reviewed atomically.
- Differential testing is straightforward.
- Early development has less repository and release overhead.
- Brokoli adapters can be developed alongside the core without coupling them
  internally.

Negative:

- The repository will contain multiple toolchains.
- CI and release automation need explicit per-language boundaries.
- Contributors may need only a subset of the full development environment.

## Rejected alternatives

- Creating one repository per SDK from the first release.
- Making SDKs thin HTTP clients that bypass the canonical IR.
