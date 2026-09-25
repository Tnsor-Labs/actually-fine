# Contract IR

The contract artifact is the stable input to the engine. IR v1.0 is described
by [`schema/contract-ir-1.0.json`](../schema/contract-ir-1.0.json).

## Canonical Form

`actually-fine` normalizes a contract before execution:

- Missing rule versions become `"1"`.
- Missing breach severity becomes `"error"`.
- Rules are sorted by rule ID.
- JSON is serialized deterministically.
- The SHA-256 digest is calculated over the canonical UTF-8 JSON bytes and
  formatted as `sha256:<64 lowercase hexadecimal characters>`.

Equivalent contracts must produce the same canonical JSON and digest even when
their object keys or rule order differ.

## Commands

```bash
actually-fine validate --contract contract.json
actually-fine inspect --contract contract.json
```

`validate` checks the artifact without reading data. `inspect` prints the
digest and canonical representation. `run` always decodes and normalizes the
contract before consuming input.

The IR version and result protocol version are independent. A contract IR
change does not implicitly change the result event schema.
