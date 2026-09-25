"""Optional Great Expectations comparison helper.

This script intentionally does not install dependencies or claim a comparison
when Great Expectations is absent. Pin the GX version and provide an equivalent
benchmark fixture before using it for published numbers.
"""

import importlib.util
import sys


if importlib.util.find_spec("great_expectations") is None:
    print("Great Expectations is not installed; native benchmark only.")
    print("Install a pinned environment separately before comparing results.")
    raise SystemExit(0)

import great_expectations as gx

print(f"Great Expectations {getattr(gx, '__version__', 'unknown')} is available.")
print("Define an equivalent NDJSON benchmark before comparing published numbers.")
