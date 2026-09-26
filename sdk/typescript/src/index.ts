import { createHash } from 'node:crypto'

const RECORD_OPERATORS = new Set(['required', 'not_null', 'type', 'format', 'regex', 'range', 'enum'])
const STREAM_OPERATORS = new Set(['unique', 'count'])
const TYPES = new Set(['string', 'number', 'integer', 'boolean', 'object', 'array'])
const ACTIONS = new Set(['warn', 'reject', 'quarantine', 'halt'])
const SEVERITIES = new Set(['warning', 'error'])

export type RuleKind = 'record' | 'stream'
export type PredicateOperator =
  | 'required'
  | 'not_null'
  | 'type'
  | 'format'
  | 'regex'
  | 'range'
  | 'enum'
  | 'unique'
  | 'count'

export interface PredicateOptions {
  type?: string
  format?: string
  pattern?: string
  min?: number
  max?: number
  values?: unknown[]
}

export interface PredicateIR extends PredicateOptions {
  op: string
}

export interface RuleOptions {
  severity?: string
  kind?: RuleKind
  version?: string
}

export interface RuleIR {
  id: string
  version: string
  kind: RuleKind
  path: string
  predicate: PredicateIR
  on_breach: { severity: string; action: string }
}

export function predicate(op: string, options: PredicateOptions = {}): PredicateIR {
  return { op, ...options }
}

export class Rule {
  constructor(
    readonly id: string,
    readonly path: string,
    readonly predicate: PredicateIR,
    readonly action: string,
    readonly options: RuleOptions = {},
  ) {}

  toIR(): RuleIR {
    if (!this.id) throw new Error('rule id is required')
    if (!RECORD_OPERATORS.has(this.predicate.op) && !STREAM_OPERATORS.has(this.predicate.op)) {
      throw new Error(`rule ${JSON.stringify(this.id)}: unsupported predicate ${JSON.stringify(this.predicate.op)}`)
    }
    const expectedKind: RuleKind = STREAM_OPERATORS.has(this.predicate.op) ? 'stream' : 'record'
    const kind = this.options.kind ?? expectedKind
    if (kind !== expectedKind) {
      throw new Error(`rule ${JSON.stringify(this.id)}: ${this.predicate.op} must be a ${expectedKind} rule`)
    }
    if (!this.path || (this.path !== '$' && !this.path.startsWith('$.'))) {
      throw new Error(`rule ${JSON.stringify(this.id)}: path must be $ or start with $.`)
    }
    const severity = this.options.severity ?? 'error'
    const version = this.options.version ?? '1'
    if (!ACTIONS.has(this.action)) throw new Error(`rule ${JSON.stringify(this.id)}: unsupported action ${JSON.stringify(this.action)}`)
    if (!SEVERITIES.has(severity)) throw new Error(`rule ${JSON.stringify(this.id)}: severity must be warning or error`)
    if (version !== '1') throw new Error(`rule ${JSON.stringify(this.id)}: unsupported version ${JSON.stringify(version)}`)

    const p = this.predicate
    if (p.op === 'type' && !TYPES.has(p.type ?? '')) throw new Error(`rule ${JSON.stringify(this.id)}: unsupported type ${JSON.stringify(p.type ?? '')}`)
    if (p.op === 'format' && p.format !== 'email') throw new Error(`rule ${JSON.stringify(this.id)}: unsupported format ${JSON.stringify(p.format ?? '')}`)
    if (p.op === 'regex') {
      if (!p.pattern) throw new Error(`rule ${JSON.stringify(this.id)}: regex pattern is required`)
      try {
        new RegExp(p.pattern)
      } catch (error) {
        throw new Error(`rule ${JSON.stringify(this.id)}: invalid regex: ${String(error)}`)
      }
    }
    if (p.op === 'range' || p.op === 'count') {
      if (p.min === undefined && p.max === undefined) throw new Error(`rule ${JSON.stringify(this.id)}: ${p.op} requires min or max`)
      if (p.min !== undefined && p.max !== undefined && p.min > p.max) throw new Error(`rule ${JSON.stringify(this.id)}: ${p.op} min cannot exceed max`)
    }
    if (p.op === 'count' && this.path !== '$') throw new Error(`rule ${JSON.stringify(this.id)}: count path must be $`)
    if (p.op === 'enum' && (!p.values || p.values.length === 0)) throw new Error(`rule ${JSON.stringify(this.id)}: enum values are required`)

    return {
      id: this.id,
      version,
      kind,
      path: this.path,
      predicate: { op: p.op, ...(p.type === undefined ? {} : { type: p.type }), ...(p.format === undefined ? {} : { format: p.format }), ...(p.pattern === undefined ? {} : { pattern: p.pattern }), ...(p.min === undefined ? {} : { min: p.min }), ...(p.max === undefined ? {} : { max: p.max }), ...(p.values === undefined ? {} : { values: p.values }) },
      on_breach: { severity, action: this.action },
    }
  }
}

export class Contract {
  readonly rules: readonly Rule[]

  constructor(readonly id: string, readonly version: string, rules: Iterable<Rule>) {
    this.rules = [...rules]
  }

  toIR() {
    if (!this.id || !this.version) throw new Error('contract id and version are required')
    if (this.rules.length === 0) throw new Error('at least one rule is required')
    const rules = this.rules.map((rule) => rule.toIR())
    const ids = new Set<string>()
    for (const rule of rules) {
      if (ids.has(rule.id)) throw new Error('rule ids must be present and unique')
      ids.add(rule.id)
    }
    rules.sort((left, right) => (left.id < right.id ? -1 : left.id > right.id ? 1 : 0))
    return {
      ir_version: '1.0',
      contract: { id: this.id, version: this.version },
      input: { kind: 'record-stream' },
      rules,
    }
  }

  canonicalJSON(): string {
    // Go's encoding/json escapes these characters even inside string values.
    return JSON.stringify(this.toIR()).replaceAll('&', '\\u0026').replaceAll('<', '\\u003c').replaceAll('>', '\\u003e')
  }

  digest(): string {
    return `sha256:${createHash('sha256').update(this.canonicalJSON(), 'utf8').digest('hex')}`
  }
}
