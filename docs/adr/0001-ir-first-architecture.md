# ADR 0001: IR-First Architecture

- Status: Accepted
- Date: 2026-09-25

## Context

actually-fine must be usable from multiple languages and integration surfaces.
SDKs must not become separate implementations of contract semantics, and the
engine must not depend on the object model of any one SDK.

Brokoli demonstrates the value of treating an intermediate representation as
the stable boundary between authoring APIs and execution. The same principle
also addresses configuration drift and migration pain: contracts can be
compiled, normalized, diffed, versioned, and replayed independently of the
authoring language.

## Decision

The canonical contract IR is the primary public interoperability boundary of
actually-fine.

SDKs are authoring frontends that compile to the IR. The CLI, engine, service
mode, and integration adapters consume the IR rather than SDK-specific
objects.

The IR will have:

- An explicit schema version.
- Canonical normalization and serialization rules.
- A stable semantic digest.
- Local structural and semantic validation.
- Compatibility and capability checks before execution.

The IR is a contract artifact. It may be committed, reviewed, diffed, signed,
cached, transported, or replayed without the original SDK source.

## Consequences

Positive:

- Python, TypeScript, Go, and future SDKs can provide equivalent behavior.
- Contract changes can be reviewed as stable artifacts.
- Runtime compatibility can fail explicitly before data is processed.
- Brokoli integration remains an adapter rather than a core dependency.

Negative:

- IR design becomes an early and significant investment.
- SDK features cannot be considered complete until their IR semantics are
  defined.
- Canonicalization and compatibility tests become part of the release bar.

## Rejected alternatives

- Treating the Go or Rust object model as the cross-language API.
- Maintaining a separate rule implementation in every SDK.
- Using YAML or JSON configuration as an informal, unversioned contract.
