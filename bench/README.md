# Benchmarks

Run the native engine benchmark with:

```bash
go test ./engine -bench BenchmarkCheck -benchmem -count=5
```

The optional Great Expectations check reports whether a GX environment is
available:

```bash
python3 bench/gx_benchmark.py
```

Published comparisons must pin versions and use equivalent rules, input data,
output behavior, and warm-up methodology. A comparison is not meaningful when
one system validates a different workload or includes unrelated setup work.
