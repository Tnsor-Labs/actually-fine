# actually-fine TypeScript SDK

The TypeScript SDK is an authoring and IR compiler surface. It does not execute
rules separately from the Go engine.

```ts
import { Contract, Rule, predicate } from 'actually-fine'

const contract = new Contract('orders', '1', [
  new Rule('email-valid', '$.email', predicate('format', { format: 'email' }), 'quarantine'),
])

contract.canonicalJSON()
contract.digest()
```

The compiler derives rule kind from the predicate, applies the engine defaults,
sorts rules by ID, and emits the same canonical IR and digest as the Go engine.
