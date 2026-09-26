import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Contract, Rule, predicate } from '../src/index.js'

test('matches the Go engine canonical digest', () => {
  const contract = new Contract('orders', '1', [
    new Rule('id-required', '$.id', predicate('required'), 'reject'),
    new Rule('email-valid', '$.email', predicate('format', { format: 'email' }), 'quarantine'),
    new Rule('amount-positive', '$.amount', predicate('range', { min: 0 }), 'reject'),
  ])
  assert.equal(contract.digest(), 'sha256:cd87b5f698ac1503f6eb622e5f560425e51260933b3be5d09e83e47dddd70503')
})

test('derives rule kind and validates stream constraints', () => {
  assert.equal(new Rule('unique-id', '$.id', predicate('unique'), 'reject').toIR().kind, 'stream')
  assert.throws(() => new Rule('unique-id', '$.id', predicate('unique'), 'reject', { kind: 'record' }).toIR(), /must be a stream rule/)
  assert.throws(() => new Rule('count', '$.id', predicate('count', { min: 1 }), 'reject').toIR(), /count path must be \$/)
})

test('normalizes defaults and sorts rules', () => {
  const contract = new Contract('x', '1', [
    new Rule('z', '$.z', predicate('required'), 'reject'),
    new Rule('a', '$.a', predicate('required'), 'reject'),
  ])
  const ir = contract.toIR()
  assert.deepEqual(ir.rules.map((rule) => rule.id), ['a', 'z'])
  assert.equal(ir.rules[0].version, '1')
  assert.equal(ir.rules[0].on_breach.severity, 'error')
})
