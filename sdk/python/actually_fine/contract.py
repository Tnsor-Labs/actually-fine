"""Author and compile actually-fine IR 1.0 contracts."""

from __future__ import annotations

from dataclasses import dataclass
import hashlib
import json
import math
import re
from typing import Any, Iterable, Optional

RECORD_OPERATORS = {"required", "not_null", "type", "format", "regex", "range", "enum"}
STREAM_OPERATORS = {"unique", "count"}
OPERATORS = RECORD_OPERATORS | STREAM_OPERATORS
TYPES = {"string", "number", "integer", "boolean", "object", "array"}
ACTIONS = {"warn", "reject", "quarantine", "halt"}
SEVERITIES = {"warning", "error"}


def _number(value: Any) -> Any:
    if isinstance(value, float) and math.isfinite(value) and value.is_integer():
        return int(value)
    if isinstance(value, list):
        return [_number(item) for item in value]
    if isinstance(value, dict):
        return {key: _number(item) for key, item in value.items()}
    return value


@dataclass(frozen=True)
class Predicate:
    op: str
    type: Optional[str] = None
    format: Optional[str] = None
    pattern: Optional[str] = None
    min: Optional[float] = None
    max: Optional[float] = None
    values: Optional[tuple[Any, ...]] = None

    def to_ir(self) -> dict[str, Any]:
        result: dict[str, Any] = {"op": self.op}
        for key in ("type", "format", "pattern", "min", "max"):
            value = getattr(self, key)
            if value is not None:
                result[key] = value
        if self.values is not None:
            result["values"] = list(self.values)
        return result


@dataclass(frozen=True)
class Rule:
    id: str
    path: str
    predicate: Predicate
    action: str
    severity: str = "error"
    kind: Optional[str] = None
    version: str = "1"

    def to_ir(self) -> dict[str, Any]:
        if not self.id:
            raise ValueError("rule id is required")
        expected_kind = "stream" if self.predicate.op in STREAM_OPERATORS else "record"
        if self.predicate.op not in OPERATORS:
            raise ValueError(f"rule {self.id!r}: unsupported predicate {self.predicate.op!r}")
        kind = self.kind or expected_kind
        if kind != expected_kind:
            raise ValueError(f"rule {self.id!r}: {self.predicate.op} must be a {expected_kind} rule")
        if not self.path or (self.path != "$" and not self.path.startswith("$.")):
            raise ValueError(f"rule {self.id!r}: path must be $ or start with $.")
        if self.action not in ACTIONS:
            raise ValueError(f"rule {self.id!r}: unsupported action {self.action!r}")
        if self.severity not in SEVERITIES:
            raise ValueError(f"rule {self.id!r}: severity must be warning or error")
        if self.version != "1":
            raise ValueError(f"rule {self.id!r}: unsupported version {self.version!r}")

        predicate = self.predicate
        if predicate.op == "type" and predicate.type not in TYPES:
            raise ValueError(f"rule {self.id!r}: unsupported type {predicate.type!r}")
        if predicate.op == "format" and predicate.format != "email":
            raise ValueError(f"rule {self.id!r}: unsupported format {predicate.format!r}")
        if predicate.op == "regex":
            if not predicate.pattern:
                raise ValueError(f"rule {self.id!r}: regex pattern is required")
            try:
                re.compile(predicate.pattern)
            except re.error as error:
                raise ValueError(f"rule {self.id!r}: invalid regex: {error}") from error
        if predicate.op in {"range", "count"}:
            if predicate.min is None and predicate.max is None:
                raise ValueError(f"rule {self.id!r}: {predicate.op} requires min or max")
            if predicate.min is not None and predicate.max is not None and predicate.min > predicate.max:
                raise ValueError(f"rule {self.id!r}: {predicate.op} min cannot exceed max")
        if predicate.op == "count" and self.path != "$":
            raise ValueError(f"rule {self.id!r}: count path must be $")
        if predicate.op == "enum" and not predicate.values:
            raise ValueError(f"rule {self.id!r}: enum values are required")

        return {
            "id": self.id,
            "version": self.version,
            "kind": kind,
            "path": self.path,
            "predicate": predicate.to_ir(),
            "on_breach": {"severity": self.severity, "action": self.action},
        }


@dataclass(frozen=True)
class Contract:
    id: str
    version: str
    rules: tuple[Rule, ...]

    def __init__(self, id: str, version: str, rules: Iterable[Rule]):
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "version", version)
        object.__setattr__(self, "rules", tuple(rules))

    def to_ir(self) -> dict[str, Any]:
        if not self.id or not self.version:
            raise ValueError("contract id and version are required")
        if not self.rules:
            raise ValueError("at least one rule is required")
        rules = [rule.to_ir() for rule in self.rules]
        ids = [rule["id"] for rule in rules]
        if len(ids) != len(set(ids)):
            raise ValueError("rule ids must be present and unique")
        rules.sort(key=lambda rule: rule["id"])
        return {
            "ir_version": "1.0",
            "contract": {"id": self.id, "version": self.version},
            "input": {"kind": "record-stream"},
            "rules": rules,
        }

    def canonical_json(self) -> bytes:
        data = _number(self.to_ir())
        encoded = json.dumps(data, ensure_ascii=False, separators=(",", ":"), allow_nan=False)
        # Match encoding/json's escaping for the characters it escapes by default.
        encoded = encoded.replace("&", "\\u0026").replace("<", "\\u003c").replace(">", "\\u003e")
        return encoded.encode("utf-8")

    def digest(self) -> str:
        return "sha256:" + hashlib.sha256(self.canonical_json()).hexdigest()
