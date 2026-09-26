package transport

// RecordBatch is the host-neutral row batch exchanged by dataset transports
// and gate adapters. Columns preserve the producer's schema order; records
// remain ordinary JSON-shaped values for contract evaluation.
type RecordBatch struct {
	Columns []string
	Records []map[string]any
}
