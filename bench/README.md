# Benchmarks

Run the native engine benchmark with:

```bash
go test ./engine -bench BenchmarkCheck -benchmem -count=5
```

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

The Great Expectations benchmark uses GX `1.23.1`'s pandas backend and the
equivalent required, email-regex, and minimum-value rules:

```bash
/tmp/actually-fine-gx-venv/bin/python bench/gx_benchmark.py --rows 100000
```

The GX input-load, setup, and warm validation costs are reported separately.
Published comparisons must pin versions and use equivalent rules, input data,
output behavior, and warm-up methodology. A comparison is not meaningful when
one system validates a different workload or includes unrelated setup work.

The first local comparison used Great Expectations `1.23.1` with its pandas
backend on 100,000 valid records and three equivalent rules. The native CLI
also parsed the NDJSON stream and routed output. Warm in-memory validation is
useful for engine comparison; end-to-end timings are useful for user
experience, but they must not be presented as the same measurement.
