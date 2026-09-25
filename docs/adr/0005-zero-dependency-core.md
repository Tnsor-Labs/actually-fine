# ADR 0005: Zero-Dependency Core and Layered Execution Modes

- Status: Accepted
- Date: 2026-09-25

## Context

The first useful data-quality check should not require a Python environment,
an orchestrator, a metadata store, a database, or a service process. Setup
complexity is one of the main reasons data-quality systems become difficult to
adopt and difficult for agents to operate.

Different users still need different integration modes. A pipeline may want an
embedded library, a language-neutral service, or a native orchestration
adapter. These modes should add capabilities without becoming prerequisites for
the core product.

## Decision

The core product will have a zero-dependency execution path, delivered as a
single CLI binary and embeddable engine.

The default path must work without:

- A Python or Node.js runtime.
- A metadata store or database.
- A service process or network connection.
- An orchestrator.
- pandas, Spark, or another dataframe runtime.

The execution layers are:

1. **CLI:** reads contract IR and data from files, stdin, or supported stream
   adapters; writes decisions and evidence to stdout or files.
2. **Embedded engine:** runs in a host application without starting a service.
3. **Service mode:** exposes the same IR and execution semantics over an
   optional protocol such as HTTP or gRPC.
4. **Adapters:** connect the engine to Brokoli and other orchestrators without
   adding their dependencies to the core.
5. **SDKs:** provide authoring convenience and compile to the canonical IR;
   they are not prerequisites for executing a contract.

The core CLI will provide stable machine-facing behavior:

- JSON Lines output for individual decisions and violations.
- Human-readable summaries as an alternate renderer.
- Exit codes that distinguish cleared data, breached data, invalid contracts,
  runtime failures, and policy failures.
- No implicit network calls or persistence.

The contract IR is the shared artifact across all layers. An optional service
or adapter must not create a second contract or result model.

## Consequences

Positive:

- A user can install one binary and run a first check immediately.
- Agents can invoke the CLI without learning a framework-specific object model.
- Brokoli and other integrations remain optional layers.
- Service and SDK users receive the same semantics as CLI users.
- Stateless local execution is easy to test and reproduce.

Negative:

- The CLI and embedded API need careful design before service mode exists.
- Stream and stateful adapters cannot all be implemented as zero-dependency
  features.
- Packaging, exit codes, and output schemas become compatibility commitments.

## Rejected alternatives

- Requiring a Python runtime for the core engine.
- Making a metadata store part of the first-run path.
- Making service mode the only supported integration surface.
- Making MCP a prerequisite for agent usage.
