import json
from pathlib import Path

import pytest

from actually_fine import Contract, Predicate, Rule


def test_compiles_to_canonical_ir_and_digest():
    contract = Contract(
        "orders",
        "1",
        [
            Rule("email-valid", "$.email", Predicate("format", format="email"), "quarantine"),
            Rule("id-required", "$.id", Predicate("required"), "reject"),
        ],
    )

    expected = json.loads(
        (Path(__file__).parents[3] / "testdata" / "orders.contract.json").read_text()
    )
    expected["rules"] = sorted(expected["rules"][:2], key=lambda rule: rule["id"])
    for rule in expected["rules"]:
        rule["version"] = "1"
    assert json.loads(contract.canonical_json()) == expected
    assert contract.digest().startswith("sha256:")
    assert len(contract.digest()) == len("sha256:") + 64


def test_digest_matches_go_engine_fixture():
    contract = Contract(
        "orders",
        "1",
        [
            Rule("id-required", "$.id", Predicate("required"), "reject"),
            Rule("email-valid", "$.email", Predicate("format", format="email"), "quarantine"),
            Rule("amount-positive", "$.amount", Predicate("range", min=0), "reject"),
        ],
    )
    assert contract.digest() == "sha256:cd87b5f698ac1503f6eb622e5f560425e51260933b3be5d09e83e47dddd70503"


def test_rule_kind_is_derived_and_cannot_be_wrong():
    assert Rule("r", "$.id", Predicate("unique"), "reject").to_ir()["kind"] == "stream"
    with pytest.raises(ValueError, match="must be a stream rule"):
        Rule("r", "$.id", Predicate("unique"), "reject", kind="record").to_ir()


def test_count_requires_root_path_and_bounds():
    with pytest.raises(ValueError, match=r"count path must be \$"):
        Rule("r", "$.id", Predicate("count", min=1), "reject").to_ir()
    with pytest.raises(ValueError, match="requires min or max"):
        Rule("r", "$", Predicate("count"), "reject").to_ir()


def test_validation_rejects_duplicate_rules_and_bad_regex():
    with pytest.raises(ValueError, match="unique"):
        Contract(
            "x",
            "1",
            [
                Rule("r", "$.a", Predicate("required"), "reject"),
                Rule("r", "$.b", Predicate("required"), "reject"),
            ],
        ).to_ir()
    with pytest.raises(ValueError, match="invalid regex"):
        Rule("r", "$.a", Predicate("regex", pattern="["), "reject").to_ir()
