# Integration Surfaces

`actually-fine` has one engine and several optional ways to reach it. The
contract IR and result protocol are the compatibility boundaries for every
surface.

```text
Python SDK     TypeScript SDK     CLI     Brokoli     MCP
      \              |              |        |        /
                canonical IR + result protocol
                              |
                         Go engine
```

## Repository Layout

The first SDKs and adapters will remain in the monorepo:

```text
cmd/actually-fine/       CLI
contract/                IR types, validation, canonicalization
engine/                  execution
result/                  result protocol
adapter/brokoli/         Brokoli integration
service/                 optional HTTP or gRPC transport
sdk/python/              Python IR compiler
sdk/typescript/          TypeScript IR compiler
agent/mcp/               optional MCP adapter
schema/                  versioned IR schemas
testdata/conformance/    shared cross-surface fixtures
```

The directories are planned boundaries. They should be added only when each
surface has a working end-to-end use case.

## Shared Conformance

Every authoring or transport surface must prove:

- Equivalent inputs produce identical canonical IR bytes.
- Equivalent contracts produce identical IR digests.
- Unsupported IR or result versions fail before execution.
- Result events match the versioned JSONL schema.
- Capability mismatches are explicit and machine-readable.

The SDK differential oracle should compare canonical output, not internal SDK
objects or implementation details.

## Brokoli Adapter

The first integration should expose a Brokoli quality gate that:

- Receives a contract IR artifact or a reference to one.
- Validates the artifact before pipeline execution.
- Streams node output through the engine.
- Maps `warn`, `reject`, `quarantine`, and `halt` to Brokoli outcomes.
- Publishes result events as run evidence.
- Does not import Brokoli types into `contract`, `engine`, or `result`.

The initial host boundary is `adapter/brokoli`. Its `Run` function accepts a
contract, an NDJSON reader, record-routing callbacks, and an evidence callback.
A Brokoli plugin owns the small conversion from its dataset/evidence APIs to
those callbacks. This is deliberately separate from Brokoli's existing
`quality_check` node: that node uses a different materialized rule model and
cannot preserve per-record evidence or the `quarantine` and `halt` actions.

For high-throughput deployments, `transport/arrow` provides the optional
`arrow-ipc/v1` codec. It is a host-neutral nested Go module so the standalone
CLI and core engine retain their dependency-light build. Brokoli is the first
consumer, while other adapters can opt into the same codec independently. Its
Arrow gate path evaluates record batches directly and emits accepted and
quarantined Arrow batches without an NDJSON conversion.

## SDKs

SDKs are compilers and authoring experiences. They may offer typed builders,
language-native helpers, and local contract validation, but they must not
silently add runtime behavior that cannot be represented in the IR.

An SDK release is ready only when its fixtures match the canonical IR and the
engine can execute its output without the SDK installed.

## Service Mode

Service mode is a transport wrapper around the engine. It must accept the same
contract artifact used by the CLI and return the same result event schema. It
may add authentication, concurrency, and state backends, but these are service
features rather than contract semantics.

## Agent Interface

MCP is an optional adapter over stable operations:

- `validate_contract`
- `inspect_contract`
- `diff_contracts`
- `run_contract`
- `explain_result`

Agents must be able to use the CLI or library directly when MCP is unavailable.
Agent-generated contracts should be staged for approval rather than silently
activated in production.
