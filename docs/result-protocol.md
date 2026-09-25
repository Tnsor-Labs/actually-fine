# Result Protocol v1.0

The CLI emits one JSON Lines event for each warning or breach. Successful
records produce no event. Human summaries are a separate rendering and are not
part of the machine protocol.

## Event

```json
{
  "result_version": "1.0",
  "status": "breach",
  "contract_id": "orders",
  "contract_version": "1",
  "rule_id": "email-valid",
  "rule_version": "1",
  "line": 2,
  "record_id": "2",
  "path": "$.email",
  "severity": "error",
  "action": "quarantine",
  "message": "value must match format \"email\"",
  "value": "bad-email"
}
```

Fields are intentionally flat and stable. `value`, `record_id`, and
`suggested_fix` are optional. Evidence must be bounded and redaction policy
will be added before sensitive data is captured by default.

`status` is `warning` when the action is `warn`, and `breach` for enforcement
actions. A warning does not produce a breach exit code. Any `reject`,
`quarantine`, or `halt` action produces exit code `1` after processing stops or
completes according to the action.

## Exit Codes

| Code | Meaning |
| ---: | --- |
| 0 | All records cleared or only warnings occurred |
| 1 | At least one enforcement breach occurred |
| 2 | Contract or command arguments are invalid |
| 3 | Input, output, or execution failed |
| 4 | An enforcement policy could not be applied |

Consumers should branch on exit code before interpreting individual events.
The result version is independent from the contract IR version and can evolve
under its own compatibility policy.
