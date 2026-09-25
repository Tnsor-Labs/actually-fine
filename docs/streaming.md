# Streaming Semantics

The MVP processes NDJSON one line at a time. It does not load the complete
input into memory. Accepted, quarantined, and result streams preserve input
order.

## Invalid Records

Malformed JSON and valid JSON values that are not objects are invalid records.
The default policy is `fail`:

```bash
actually-fine run --contract contract.json --input events.ndjson
```

The command writes an error to stderr and returns exit code `3`. The invalid
line is never written to accepted output.

To continue while preserving the malformed line for review, use:

```bash
actually-fine run \
  --contract contract.json \
  --input events.ndjson \
  --invalid-record quarantine \
  --quarantine-output rejected.ndjson
```

This emits a versioned `input.parse` breach event and writes the original line
to quarantine. There is no silent skip mode.

## Actions

- `warn` records a warning and keeps the record in accepted output.
- `reject` drops the record and continues.
- `quarantine` writes the original record to quarantine and continues.
- `halt` records the breach, preserves all output already written, and stops
  before reading the next record.

All writes are synchronous. A failed output write, broken pipe, scanner error,
or interrupted execution returns exit code `3`; these are runtime failures, not
data breaches.

The scanner accepts records up to 16 MiB. Larger lines fail explicitly instead
of causing unbounded memory growth.
