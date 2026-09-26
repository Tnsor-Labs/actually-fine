# Arrow IPC Transport

This optional nested module reads and writes the `arrow-ipc/v1` dataset
transport. It maps Arrow values into the same logical record types used by the
engine and preserves integer identifiers exactly, including values above
`2^53`. Brokoli is the first consumer, but the transport is host-neutral.

Run its tests from this directory:

```bash
go test ./...
```

The root module intentionally does not include Arrow, so the standalone CLI
and core engine remain dependency-light:

```bash
go test ./...
```
