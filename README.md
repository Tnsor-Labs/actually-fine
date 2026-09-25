# actually-fine

**Executable data contracts for real pipelines.**

`actually-fine` is an open-source data quality and contract engine for
enforcing data rules as data moves through a system.

It is designed to run at pipeline boundaries: on a record, a stream, a
micro-batch, or a full batch. A violation is not only a report. It can become
an operational decision: continue, warn, reject, quarantine, reroute, or halt.

The project is standalone, with a first-class Brokoli integration. Brokoli is
an important integration target, not a dependency or boundary for the engine.

> We do not check expectations after the fact. We enforce contracts as data
> moves.

## Status

Early MVP. The repository now contains a working Go engine and CLI for
versioned contracts over newline-delimited JSON. The remaining work is to
expand the contract model, harden the output compatibility guarantees, and add
integrations.

The project is being built in this order:

1. Define and validate the contract intermediate representation (IR).
2. Build and test the engine against the IR.
3. Prove the core workflow through the CLI.
4. Add language SDKs after the engine is a useful, stable product.
5. Add integrations, including the Brokoli adapter.

The API, IR, rule vocabulary, and implementation language are not all final.
See [`docs/adr/`](docs/adr/) for decisions and open questions.

## Quick Start

Build the binary:

```bash
go build -o actually-fine ./cmd/actually-fine
```

Validate an NDJSON stream:

```bash
./actually-fine run \
  --contract testdata/orders.contract.json \
  --input testdata/orders.ndjson \
  --valid-output accepted.ndjson \
  --quarantine-output rejected.ndjson \
  --results violations.jsonl
```

The CLI writes accepted records and quarantined records to separate streams,
and returns a non-zero status when the input breaches the contract. Use
`--format jsonl` when the caller needs breach events on stdout instead of a
human summary.

The current MVP supports required fields, types, email format, regular
expressions, numeric ranges, and enum membership. Supported actions are
`warn`, `reject`, `quarantine`, and `halt`.

Inspect or validate a contract artifact without reading data:

```bash
./actually-fine validate --contract testdata/orders.contract.json
./actually-fine inspect --contract testdata/orders.contract.json
```

`validate` checks the IR and prints its stable digest. `inspect` prints the
canonical normalized JSON used to calculate that digest.

## Why

Data quality tooling should not require a platform before it can validate one
field. `actually-fine` is being designed as a smaller, more direct alternative
to systems that introduce multiple framework abstractions, configuration
stores, and batch-specific execution models before producing a useful result.

The project specifically addresses these failure modes:

- **Conceptual overload:** one contract model instead of contexts, suites,
  checkpoints, batch requests, and datasource objects.
- **Configuration drift:** code-first authoring compiled to a canonical IR,
  rather than a second YAML or JSON source of truth.
- **Migration pain:** explicit IR versions, compatibility checks, and a small
  public surface intended to remain stable.
- **Batch-only validation:** one execution model for records, streams, windows,
  micro-batches, and batches.
- **Unusable output:** flat, predictable violation events designed for logs,
  scripts, and AI agents.
- **Runtime overhead:** lightweight record and stream processing without a
  mandatory pandas, Spark, or dataframe runtime.
- **Stale reporting:** direct terminal output and event sinks instead of a
  required generated documentation site.

## Core Ideas

### The IR is the product boundary

Language SDKs will be authoring frontends, not separate implementations of
contract semantics. They will compile to the same versioned, canonical IR.

The IR can be committed, reviewed, diffed, signed, cached, transported, and
replayed without the original SDK source.

See [`docs/contract-ir.md`](docs/contract-ir.md) and the
[IR v1.0 schema](schema/contract-ir-1.0.json) for the current artifact
definition.

```text
authoring API -> canonical contract IR -> engine -> decision and evidence
```

This makes contracts portable across the CLI, service mode, Brokoli, and
future integrations.

### Enforcement is part of the contract

A rule describes both what is valid and what happens when it is not:

```text
email must be valid
  breach: error
  action: quarantine
  continue: true
```

The engine should make the boundary decision directly instead of requiring a
second orchestration layer to interpret a test report.

### Portable rules first

Portable declarative rules will have engine-defined semantics. Examples include
nullability, types, ranges, formats, uniqueness, freshness, reference
existence, and stream-window constraints.

Custom behavior may be supported through explicit runtime extensions with a
declared execution target. Arbitrary language closures will not be silently
treated as portable rules.

### Stateless by default

A single check should not require a metadata store. Stateful rules can declare
their state requirements and receive an explicit state backend when needed.

## Results

Results are designed to be useful in a terminal and directly consumable by
machines:

```json
{
  "status": "breach",
  "contract": "customer-events",
  "rule": "email-valid",
  "severity": "error",
  "action": "quarantine",
  "field": "email",
  "value": "not-an-email",
  "message": "email is not a valid address",
  "record_id": "42"
}
```

The result schema is versioned independently from the contract IR.

The current result protocol is documented in
[`docs/result-protocol.md`](docs/result-protocol.md). It defines versioned,
flat JSONL events and stable process exit codes.

Streaming behavior, malformed-record policies, action semantics, and output
ordering are documented in [`docs/streaming.md`](docs/streaming.md).

## Benchmarks

The native engine benchmark is under [`bench/`](bench/). Run it with:

```bash
go test ./engine -bench BenchmarkCheck -benchmem -count=5
```

An optional Great Expectations comparison helper reports whether a pinned GX
environment is available. Comparisons should use equivalent rules and input
workloads; the current environment does not include Great Expectations.

## Planned Interfaces

The initial product is the engine and CLI. Future interfaces may include:

- Go library and embeddable engine.
- CLI for files, streams, and contract artifacts.
- HTTP or gRPC service mode.
- Native Brokoli adapter.
- Python and TypeScript SDKs compiled to the same IR.
- Agent-callable interface, potentially through MCP.

SDK work follows the engine. An SDK without a proven engine and product
workflow would only multiply unstable assumptions.

## Repository Structure

The engine, CLI, IR schema, conformance fixtures, and future SDKs will live in
this repository:

```text
github.com/Tnsor-Labs/actually-fine
```

All SDKs will eventually share canonicalization, validation, digest, and
differential fixtures. They will not be runtime dependencies of the engine.

## Architecture Decisions

Read the [Architecture Decision Records](docs/adr/README.md), beginning with:

- [IR-first architecture](docs/adr/0001-ir-first-architecture.md)
- [Single repository for SDKs](docs/adr/0002-single-repository-for-sdks.md)
- [Portable rules and runtime extensions](docs/adr/0003-portable-rules-and-extensions.md)
- [Engine implementation language](docs/adr/0004-engine-implementation-language.md)
- [Zero-dependency core](docs/adr/0005-zero-dependency-core.md)
- [MVP scope and engine selection](docs/adr/0006-mvp-scope-and-engine.md)

## Contributing

The project is at the architecture stage. Product and design discussions are
welcome before implementation expands. New architectural decisions should be
recorded as ADRs, and proposed behavior should not be described as available
until it is implemented and tested.

## License

The intended project model is free and open source. The license is still an
explicit project decision and has not yet been selected.
