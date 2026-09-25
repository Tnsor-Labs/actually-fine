# Rule Catalog

The MVP uses structured predicates in the contract IR. Rules do not contain
language-specific code or string expressions.

## Record Rules

| Predicate | Meaning |
| --- | --- |
| `required` | Path exists, is not null, and is not an empty string |
| `not_null` | Path exists and is not null; empty strings are allowed |
| `type` | Value has the exact JSON type requested |
| `format` | Value matches a supported format, currently `email` |
| `regex` | String value matches the supplied regular expression |
| `range` | Numeric value is between optional `min` and `max`, inclusive |
| `enum` | Value exactly matches one of the JSON values in `values` |

Missing paths and explicit JSON null are distinct in diagnostics. Numeric
validation is strict: strings such as `"42"` are not coerced into numbers.
Integer values must be JSON numbers with no fractional component.

## Stream Rules

Stream rules maintain state for one execution and are evaluated without loading
the complete input into memory.

| Predicate | Meaning |
| --- | --- |
| `unique` | Every non-null value at the path occurs once; missing and null values breach |
| `count` | The total number of parsed object records meets `min` and `max` |

`count` uses path `$` and emits its result after the input stream completes.
If execution halts or fails before completion, final stream rules are not
evaluated because the stream is incomplete.

Every rule has an action and severity. Actions determine routing; severity
describes the finding:

- `warn` keeps the record in accepted output.
- `reject` drops the record.
- `quarantine` writes the record to quarantine.
- `halt` stops before reading the next record.
