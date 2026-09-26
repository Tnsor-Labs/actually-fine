# Release Process

actually-fine releases are tag-driven and independently versioned from
downstream integrations.

## Release Steps

1. Confirm `main` is green in CI and the intended changes are listed under
   `CHANGELOG.md`'s `[Unreleased]` section.
2. Review the version as a semantic-version increment. Use an RC tag such as
   `v0.1.0-rc1` when a stabilization build is useful.
3. Tag the release from the reviewed commit and push the tag:

   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

4. The `Release` workflow runs the full Go test suite and GoReleaser. It
   publishes platform archives, checksums, and the repository README and
   license.
5. Replace GoReleaser's generated release body with curated notes:

   ```bash
   gh release edit v0.1.0 --notes-file release-notes.md
   ```

6. Move `[Unreleased]` into a dated version section in `CHANGELOG.md` in a
   follow-up change, preserving the same release-note content.

## Release Notes

Generated commit lists are a fallback, not the product release notes. Curated
notes should:

- Group changes by area, leading with why each change matters.
- Name the owner of every entry by GitHub handle.
- Include a **Before you upgrade** section for behavior or compatibility
  changes. If there are none, say so explicitly.
- Mention new contract IR, result protocol, SDK, transport, and integration
  surfaces when they are part of the release.
- Include a short orientation paragraph when the release changes the project
  shape enough that new contributors need context.

## Release Artifacts

Each release contains Linux, macOS, and Windows archives for amd64 and arm64,
plus `checksums.txt`. Archives include `README.md` and `LICENSE`. Snapshot
builds are for local verification only and are not published.
