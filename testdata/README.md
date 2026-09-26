# Test Data

Generate a deterministic synthetic orders pipeline workload:

```bash
python3 testdata/generate_synthetic_pipeline.py \
  --rows 100000 \
  --output /tmp/actually-fine-synthetic
```

The workload includes malformed JSON, missing and null nested fields, invalid
email values, negative amounts, numeric strings, duplicate order IDs, and
unknown statuses. Run it through the CLI with:

```bash
actually-fine run \
  --contract /tmp/actually-fine-synthetic/contract.json \
  --input /tmp/actually-fine-synthetic/orders.ndjson \
  --invalid-record quarantine \
  --valid-output /tmp/actually-fine-synthetic/accepted.ndjson \
  --quarantine-output /tmp/actually-fine-synthetic/quarantine.ndjson \
  --results /tmp/actually-fine-synthetic/results.jsonl
```

The command returns exit code `1` because the generated input intentionally
contains contract breaches. Use `--invalid-record fail` to verify fail-fast
behavior on malformed input; the generated workload begins with malformed JSON
and should return exit code `3`.
