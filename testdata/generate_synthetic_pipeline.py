"""Generate a deterministic synthetic orders pipeline workload."""

import argparse
import json
from pathlib import Path


def contract():
    return {
        "ir_version": "1.0",
        "contract": {"id": "synthetic-orders", "version": "1"},
        "input": {"kind": "record-stream"},
        "rules": [
            {
                "id": "order-id-required",
                "kind": "record",
                "path": "$.order_id",
                "predicate": {"op": "required"},
                "on_breach": {"severity": "error", "action": "reject"},
            },
            {
                "id": "order-id-unique",
                "kind": "stream",
                "path": "$.order_id",
                "predicate": {"op": "unique"},
                "on_breach": {"severity": "error", "action": "quarantine"},
            },
            {
                "id": "email-valid",
                "kind": "record",
                "path": "$.customer.email",
                "predicate": {"op": "format", "format": "email"},
                "on_breach": {"severity": "error", "action": "quarantine"},
            },
            {
                "id": "amount-number",
                "kind": "record",
                "path": "$.amount",
                "predicate": {"op": "type", "type": "number"},
                "on_breach": {"severity": "error", "action": "reject"},
            },
            {
                "id": "amount-nonnegative",
                "kind": "record",
                "path": "$.amount",
                "predicate": {"op": "range", "min": 0},
                "on_breach": {"severity": "error", "action": "quarantine"},
            },
            {
                "id": "status-known",
                "kind": "record",
                "path": "$.status",
                "predicate": {"op": "enum", "values": ["active", "cancelled", "pending"]},
                "on_breach": {"severity": "warning", "action": "warn"},
            },
            {
                "id": "minimum-orders",
                "kind": "stream",
                "path": "$",
                "predicate": {"op": "count", "min": 99000},
                "on_breach": {"severity": "error", "action": "reject"},
            },
        ],
    }


def record(index):
    order_id = f"order-{index:06d}"
    if index % 211 == 0:
        order_id = f"order-{index - 1:06d}"
    email = f"customer{index}@example.com"
    if index % 149 == 0:
        email = "not-an-email"
    customer = {"email": email}
    if index % 113 == 0:
        customer.pop("email")
    elif index % 127 == 0:
        customer["email"] = None
    amount = float((index % 1000) + 1)
    if index % 173 == 0:
        amount = -amount
    if index % 191 == 0:
        amount = str(amount)
    status = "active"
    if index % 223 == 0:
        status = "unknown"
    return {"order_id": order_id, "customer": customer, "amount": amount, "status": status}


parser = argparse.ArgumentParser()
parser.add_argument("--rows", type=int, default=100_000)
parser.add_argument("--output", type=Path, required=True)
args = parser.parse_args()

args.output.mkdir(parents=True, exist_ok=True)
(args.output / "contract.json").write_text(json.dumps(contract(), indent=2) + "\n")
with (args.output / "orders.ndjson").open("w") as output:
    for index in range(args.rows):
        if index % 997 == 0:
            output.write("{malformed-json\n")
        else:
            output.write(json.dumps(record(index), separators=(",", ":")) + "\n")

print(args.output)
