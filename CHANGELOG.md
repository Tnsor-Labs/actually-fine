# Changelog

All notable changes to actually-fine are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and releases follow
[Semantic Versioning](https://semver.org/).

Every entry names the owner of the work. Release notes explain the operational
impact rather than repeating commit subjects.

## [Unreleased]

### Added

- **Python and TypeScript authoring SDKs** -- @hc12r. Both compile contracts
  to the canonical IR without embedding a second rule engine, and both produce
  the same digest as the Go engine.
- **Arrow batch transport** -- @hc12r. High-throughput integrations can gate
  accepted and quarantined Arrow batches without converting through NDJSON.
- **Brokoli contract gate boundary** -- @hc12r. The engine can be embedded at
  a Brokoli pipeline boundary while keeping host-specific types outside the
  core packages.

### Changed

- **Release artifacts now carry build metadata** -- @hc12r. Binaries expose
  their version, commit, and build date through `actually-fine version`.

### Fixed

- **Go numeric values are handled consistently by the engine** -- @hc12r.
  Numeric inputs from all supported Go numeric types now follow the same rule
  semantics.

## Release notes

GoReleaser publishes a generated commit list as a fallback. Curated release
notes are added to the GitHub release after the workflow completes and are
mirrored here when a version is cut. See [`RELEASING.md`](RELEASING.md).
