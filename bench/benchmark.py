"""Run reproducible native and optional Great Expectations benchmarks."""

import argparse
import hashlib
import json
import os
import platform
import re
import statistics
import subprocess
import tempfile
import time
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def write_workload(path, rows, invalid_every):
    digest = hashlib.sha256()
    with path.open("wb") as output:
        for index in range(rows):
            record = {
                "id": str(index),
                "email": "invalid" if invalid_every and index % invalid_every == 0 else f"person{index}@example.com",
                "amount": float(index % 1000),
            }
            line = (json.dumps(record, separators=(",", ":")) + "\n").encode()
            output.write(line)
            digest.update(line)
    return digest.hexdigest()


def native_sample(binary, contract, input_path, expected_exit):
    command = [binary,
        "run",
        "--contract",
        str(contract),
        "--input",
        str(input_path),
        "--valid-output",
        os.devnull,
        "--results",
        os.devnull,
    ]
    timed_command = command
    if Path("/usr/bin/time").exists():
        timed_command = ["/usr/bin/time", "-f", "%M"] + command
    start = time.perf_counter()
    completed = subprocess.run(timed_command, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)
    elapsed = time.perf_counter() - start
    if completed.returncode != expected_exit:
        raise RuntimeError(f"native benchmark failed ({completed.returncode}): {completed.stderr}")
    peak_rss = None
    try:
        peak_rss = int(completed.stderr.strip().splitlines()[-1])
    except (ValueError, IndexError):
        pass
    return {"seconds": elapsed, "peak_rss_kb": peak_rss}


def run_gx(python, input_path, iterations):
    command = [
        python,
        str(ROOT / "bench" / "gx_benchmark.py"),
        "--input",
        str(input_path),
        "--iterations",
        str(iterations),
        "--json",
    ]
    completed = subprocess.run(command, capture_output=True, text=True, check=True)
    return json.loads(completed.stdout)


def run_gx_cold(python, input_path):
    command = [
        python,
        str(ROOT / "bench" / "gx_benchmark.py"),
        "--input",
        str(input_path),
        "--iterations",
        "1",
        "--json",
    ]
    start = time.perf_counter()
    completed = subprocess.run(command, capture_output=True, text=True, check=True)
    elapsed = time.perf_counter() - start
    return elapsed, json.loads(completed.stdout)


def run_go_benchmark():
    command = [
        "go",
        "test",
        "./engine",
        "-run",
        "^$",
        "-bench",
        "^BenchmarkCheck$",
        "-benchmem",
        "-count=5",
    ]
    completed = subprocess.run(command, cwd=ROOT, capture_output=True, text=True, check=True)
    samples = []
    for line in completed.stdout.splitlines():
        match = re.search(r"BenchmarkCheck-\d+\s+\d+\s+([0-9.]+) ns/op\s+([0-9.]+) B/op\s+(\d+) allocs/op", line)
        if match:
            samples.append({"ns_per_record": float(match.group(1)), "bytes_per_record": float(match.group(2)), "allocs_per_record": int(match.group(3))})
    if not samples:
        raise RuntimeError(f"could not parse Go benchmark output: {completed.stdout}")
    median_ns = statistics.median(sample["ns_per_record"] for sample in samples)
    return {
        "samples": samples,
        "median_ns_per_record": median_ns,
        "records_per_second": 1_000_000_000 / median_ns,
    }


parser = argparse.ArgumentParser()
parser.add_argument("--binary", default=os.environ.get("ACTUALLY_FINE_BIN", "./actually-fine"))
parser.add_argument("--gx-python", default=os.environ.get("GX_PYTHON"))
parser.add_argument("--rows", type=int, default=100_000)
parser.add_argument("--iterations", type=int, default=5)
parser.add_argument("--invalid-every", type=int, default=0, help="make every Nth email invalid")
parser.add_argument("--output", type=Path)
args = parser.parse_args()

contract = ROOT / "testdata" / "orders.contract.json"
with tempfile.TemporaryDirectory(prefix="actually-fine-bench-") as temporary:
    input_path = Path(temporary) / "input.ndjson"
    input_hash = write_workload(input_path, args.rows, args.invalid_every)
    expected_exit = 1 if args.invalid_every else 0
    native_samples = [native_sample(args.binary, contract, input_path, expected_exit) for _ in range(args.iterations)]
    result = {
        "workload": {
            "rows": args.rows,
            "rules": 3,
            "invalid_every": args.invalid_every,
            "input_sha256": input_hash,
            "contract": str(contract.relative_to(ROOT)),
        },
        "environment": {
            "os": platform.platform(),
            "machine": platform.machine(),
            "python": platform.python_version(),
            "go": subprocess.run(["go", "version"], capture_output=True, text=True, check=True).stdout.strip(),
        },
        "native_engine": run_go_benchmark(),
        "native_cli": {
            "samples_seconds": [sample["seconds"] for sample in native_samples],
            "median_seconds": statistics.median(sample["seconds"] for sample in native_samples),
            "rows_per_second": args.rows / statistics.median(sample["seconds"] for sample in native_samples),
            "peak_rss_kb": [sample["peak_rss_kb"] for sample in native_samples],
        },
    }
    if args.gx_python:
        gx_cold_seconds, gx_cold = run_gx_cold(args.gx_python, input_path)
        gx_warm = run_gx(args.gx_python, input_path, args.iterations)
        gx_warm["cold_process_seconds"] = gx_cold_seconds
        gx_warm["cold_warm_validation_seconds"] = gx_cold["warm_median_seconds"]
        result["great_expectations"] = gx_warm
    else:
        result["great_expectations"] = None

    rendered = json.dumps(result, indent=2, sort_keys=True)
    if args.output:
        args.output.write_text(rendered + "\n")
    print(rendered)
