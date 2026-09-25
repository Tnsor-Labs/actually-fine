# Benchmarks

Run the native in-process engine benchmark with:

```bash
go test ./engine -bench BenchmarkCheck -benchmem -count=5
```

This isolates rule execution from JSON parsing, process startup, and output.

Run the reproducible end-to-end suite after building the binary:

```bash
go build -o /tmp/actually-fine ./cmd/actually-fine
GX_PYTHON=/tmp/actually-fine-gx-venv/bin/python \
  python3 bench/benchmark.py \
  --binary /tmp/actually-fine \
  --rows 100000 \
  --iterations 5 \
  --output /tmp/actually-fine-benchmark.json
```

Run the equivalent invalid-data workload by adding
`--invalid-every 100` (one invalid email per 100 records). The runner expects
the native breach exit code and records GX's unsuccessful validation result.

The runner generates one deterministic NDJSON workload and records its SHA-256
hash. It reports native engine microbenchmarks, native cold CLI samples,
median throughput, and peak RSS. If `GX_PYTHON` is supplied, it also runs the
pinned GX pandas benchmark against the same file.

For an end-to-end CLI measurement over a large NDJSON stream, build the binary
and time the same contract and input used by the GX benchmark:

```bash
go build -o /tmp/actually-fine ./cmd/actually-fine
/usr/bin/time -f 'native_seconds=%e' /tmp/actually-fine run \
  --contract testdata/orders.contract.json \
  --input /tmp/actually-fine-100k.ndjson \
  --valid-output /dev/null \
  --results /dev/null >/dev/null
```

The standalone Great Expectations benchmark uses GX `1.23.1`'s pandas backend
and the equivalent required, email-regex, and minimum-value rules:

```bash
/tmp/actually-fine-gx-venv/bin/python bench/gx_benchmark.py --rows 100000
```

The GX input-load, setup, and warm validation costs are reported separately.
Published comparisons must pin versions and use equivalent rules, input data,
output behavior, and warm-up methodology. A comparison is not meaningful when
one system validates a different workload or includes unrelated setup work.

Warm in-memory validation is useful for engine comparison; cold end-to-end
timings are useful for user experience. They are separate measurements and
must not be presented as one unsupported headline number.
