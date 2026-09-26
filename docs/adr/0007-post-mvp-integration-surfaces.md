# ADR 0007: Post-MVP Integration Surfaces

- Status: Accepted
- Date: 2026-09-26

## Context

The engine is useful on its own, but adoption requires integrations for
different authoring and execution environments. These integrations must not
reintroduce the coupling and installation burden that the zero-dependency core
is intended to remove.

The canonical contract IR and result protocol now provide stable boundaries.
SDKs, service mode, Brokoli, and agent tooling should build on those
boundaries rather than define parallel semantics.

## Decision

Post-MVP integrations will be layered in this order:

1. **Brokoli adapter:** a native gate/node adapter that carries contract IR,
   maps result events to pipeline outcomes, and uses Brokoli-specific types
   only outside the core engine.
2. **Python SDK:** an authoring API that compiles contracts to canonical IR;
   it does not execute a second Python implementation of the rules.
3. **TypeScript SDK:** an equivalent IR compiler with differential fixtures
   against the Python SDK and canonical examples.
4. **Service mode:** optional HTTP or gRPC transport for the same IR and result
   protocol used by the CLI.
5. **Agent interface:** an optional MCP adapter over contract inspection,
   validation, execution, result explanation, and contract diff operations.

The monorepo will use separate package boundaries for each surface. Every
surface must pass the shared IR and result conformance suites. No surface may
add a hard dependency to the zero-dependency CLI or embedded engine.

Capability negotiation is required for remote execution. A client must be able
to discover supported IR versions, rule predicates, result versions, input
types, and execution features before submitting work. Unsupported capability
requests fail before data is processed.

The SDKs and agent interface will be developed only after the engine and CLI
contracts are stable enough to serve as their test oracle.

## Consequences

Positive:

- Every integration shares one semantic implementation.
- Python and TypeScript users receive ergonomic authoring without runtime
  lock-in.
- Brokoli remains first-class without becoming a core dependency.
- Agents can use stable CLI operations even when MCP is unavailable.
- Remote clients get explicit compatibility failures.

Negative:

- Integration work is intentionally delayed until the core contracts mature.
- Conformance fixtures and capability documents become release requirements.
- Each ecosystem needs packaging, release, and compatibility maintenance.

## Rejected alternatives

- Executing Python or TypeScript rules independently from the engine.
- Making SDKs required to run a contract.
- Making MCP the primary execution protocol.
- Importing Brokoli packages into the core engine.
