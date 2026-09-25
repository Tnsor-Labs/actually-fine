"""Benchmark the Great Expectations pandas path for the MVP rule workload."""

import argparse
import importlib.util
import json
import statistics
import time


if importlib.util.find_spec("great_expectations") is None:
    print("Great Expectations is not installed; native benchmark only.")
    print("Install a pinned environment separately before comparing results.")
    raise SystemExit(0)

import great_expectations as gx
import pandas as pd


def make_batch(dataframe):
    context = gx.get_context(mode="ephemeral")
    context._project_config.progress_bars = {
        "globally": False,
        "metric_calculations": False,
    }
    datasource = context.data_sources.add_pandas(name="benchmark")
    asset = datasource.add_dataframe_asset(name="orders")
    definition = asset.add_batch_definition_whole_dataframe("whole")
    return definition.get_batch(batch_parameters={"dataframe": dataframe})


def validate(batch, expectations):
    return [
        batch.validate(expectation, result_format="BOOLEAN_ONLY")
        for expectation in expectations
    ]


parser = argparse.ArgumentParser()
parser.add_argument("--input", help="NDJSON input file; otherwise generate rows")
parser.add_argument("--rows", type=int, default=100_000)
parser.add_argument("--iterations", type=int, default=5)
parser.add_argument("--json", action="store_true", help="emit one JSON result")
args = parser.parse_args()

load_start = time.perf_counter()
if args.input:
    dataframe = pd.read_json(args.input, lines=True)
else:
    dataframe = pd.DataFrame(
        {
            "id": range(args.rows),
            "email": [f"person{i}@example.com" for i in range(args.rows)],
            "amount": [float(i % 1000) for i in range(args.rows)],
        }
    )
load_seconds = time.perf_counter() - load_start

expectations = [
    gx.expectations.ExpectColumnValuesToNotBeNull(column="id"),
    gx.expectations.ExpectColumnValuesToMatchRegex(
        column="email", regex=r"^[^@\s]+@[^@\s]+\.[^@\s]+$"
    ),
    gx.expectations.ExpectColumnValuesToBeBetween(column="amount", min_value=0),
]

setup_start = time.perf_counter()
batch = make_batch(dataframe)
setup_seconds = time.perf_counter() - setup_start

timings = []
for _ in range(args.iterations):
    start = time.perf_counter()
    results = validate(batch, expectations)
    timings.append(time.perf_counter() - start)

median_seconds = statistics.median(timings)
output = {
    "gx_version": gx.__version__,
    "backend": "pandas",
    "rows": len(dataframe),
    "rules": len(expectations),
    "load_seconds": load_seconds,
    "setup_seconds": setup_seconds,
    "warm_samples_seconds": timings,
    "warm_median_seconds": median_seconds,
    "warm_rows_per_second": len(dataframe) / median_seconds,
    "success": all(result.success for result in results),
}
if args.json:
    print(json.dumps(output, sort_keys=True))
else:
    print(f"Great Expectations {gx.__version__}")
    print(f"backend=pandas rows={len(dataframe)} rules={len(expectations)}")
    print(f"load_seconds={load_seconds:.6f}")
    print(f"setup_seconds={setup_seconds:.6f}")
    print(f"warm_median_seconds={median_seconds:.6f}")
    print(f"warm_rows_per_second={len(dataframe) / median_seconds:.0f}")
    print(f"success={all(result.success for result in results)}")
