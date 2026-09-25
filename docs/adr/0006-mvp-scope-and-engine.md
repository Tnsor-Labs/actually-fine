# ADR 0006: MVP Scope and Engine Selection

- Status: Accepted
- Date: 2026-09-25

## Context

The project needs a working product before adding SDKs, service mode, agent
interfaces, or orchestrator integrations. The first product must prove that a
small, zero-dependency engine can enforce contracts on a live stream and route
breaches into useful outputs.

## Decision

The MVP will be implemented in Go as a single CLI and embeddable library.

The MVP supports:

- Versioned JSON contract IR.
- Newline-delimited JSON input from a file or stdin.
- Record-level rules: required fields, types, formats, regex, numeric ranges,
  and enum membership.
- Actions: `warn`, `reject`, `quarantine`, and `halt`.
- Accepted and quarantined output streams.
- Human summaries and JSON Lines result events.
- Deterministic behavior and stable exit codes.

The MVP does not include SDKs, a service, MCP, database connectors, CSV, stateful
windows, reference data, custom code rules, or a web interface.

The initial benchmark compares the native engine's throughput, memory, and
latency using the same NDJSON workload. Great Expectations comparison is
optional and must be run in a separately declared environment with pinned
versions and equivalent validation semantics.

## Consequences

Positive:

- The project can demonstrate its core product quickly.
- Go provides a small, distributable binary aligned with Brokoli.
- The benchmark measures the actual zero-dependency path.
- SDK and integration decisions are informed by a working engine.

Negative:

- JSON IR is initially less ergonomic than a future SDK.
- NDJSON does not cover every pipeline source.
- Some GX comparisons will not be exactly equivalent until broader rule and
  dataset support exists.
