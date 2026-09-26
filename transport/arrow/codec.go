// Package arrow provides the optional Arrow IPC transport. It is kept below
// the transport boundary so contract and engine remain independent of Arrow;
// Brokoli is one consumer of this package, not its owner.
package arrow

import (
	"context"
	"fmt"
	"io"
	"math"

	arrowgo "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"

	"github.com/Tnsor-Labs/actually-fine/engine"
	"github.com/Tnsor-Labs/actually-fine/transport"
)

const Codec = "arrow-ipc/v1"

type Dataset struct {
	Columns []string
	Rows    []engine.Record
}

// BatchReader exposes Arrow record batches without an intermediate NDJSON
// representation. The returned values use the same logical value mapping as
// Decode.
type BatchReader struct {
	reader  *ipc.Reader
	columns []string
}

func NewBatchReader(r io.Reader) (*BatchReader, error) {
	reader, err := ipc.NewReader(r, ipc.WithAllocator(memory.DefaultAllocator))
	if err != nil {
		return nil, fmt.Errorf("open arrow ipc stream: %w", err)
	}
	columns := make([]string, 0, len(reader.Schema().Fields()))
	for _, field := range reader.Schema().Fields() {
		columns = append(columns, field.Name)
	}
	return &BatchReader{reader: reader, columns: columns}, nil
}

func (r *BatchReader) Next(ctx context.Context) (transport.RecordBatch, error) {
	if err := ctx.Err(); err != nil {
		return transport.RecordBatch{}, err
	}
	if !r.reader.Next() {
		if err := r.reader.Err(); err != nil && err != io.EOF {
			return transport.RecordBatch{}, fmt.Errorf("read arrow ipc stream: %w", err)
		}
		return transport.RecordBatch{}, io.EOF
	}
	record := r.reader.Record()
	rows := make([]map[string]any, 0, record.NumRows())
	for row := int64(0); row < record.NumRows(); row++ {
		out := make(map[string]any, len(r.columns))
		for column, values := range record.Columns() {
			value, err := valueAt(values, int(row))
			if err != nil {
				return transport.RecordBatch{}, fmt.Errorf("column %q row %d: %w", r.columns[column], len(rows), err)
			}
			out[r.columns[column]] = value
		}
		rows = append(rows, out)
	}
	return transport.RecordBatch{Columns: append([]string(nil), r.columns...), Records: rows}, nil
}

func (r *BatchReader) Close() error {
	r.reader.Release()
	return nil
}

// BatchWriter emits one Arrow IPC stream containing all batches written to
// it. The first batch establishes the output schema; later batches must use
// the same columns and compatible logical value types.
type BatchWriter struct {
	writer *ipc.Writer
	output io.Writer
	schema *arrowgo.Schema
	fields []string
	kinds  []string
}

func NewBatchWriter(w io.Writer) *BatchWriter {
	return &BatchWriter{writer: nil, output: w}
}

func (w *BatchWriter) Write(ctx context.Context, batch transport.RecordBatch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if w.writer == nil {
		if w.output == nil {
			return fmt.Errorf("arrow batch writer has no output")
		}
		w.fields = append([]string(nil), batch.Columns...)
		w.kinds = make([]string, len(w.fields))
		fields := make([]arrowgo.Field, len(w.fields))
		for i, column := range w.fields {
			kind := "string"
			for _, row := range batch.Records {
				if value, ok := row[column]; ok && value != nil {
					kind = valueKind(value)
					break
				}
			}
			w.kinds[i] = kind
			fields[i] = arrowgo.Field{Name: column, Type: typeFor(kind), Nullable: true}
		}
		schema := arrowgo.NewSchema(fields, nil)
		w.schema = schema
		w.writer = ipc.NewWriter(w.output, ipc.WithSchema(schema))
	} else if len(batch.Columns) != len(w.fields) {
		return fmt.Errorf("arrow batch columns changed from %d to %d", len(w.fields), len(batch.Columns))
	} else {
		for i := range w.fields {
			if batch.Columns[i] != w.fields[i] {
				return fmt.Errorf("arrow batch column %d changed from %q to %q", i, w.fields[i], batch.Columns[i])
			}
		}
	}

	builder := array.NewRecordBuilder(memory.DefaultAllocator, w.schema)
	defer builder.Release()
	for _, row := range batch.Records {
		for column, name := range w.fields {
			appendValue(builder.Field(column), w.kinds[column], row[name])
		}
	}
	record := builder.NewRecord()
	defer record.Release()
	if err := w.writer.Write(record); err != nil {
		return fmt.Errorf("write arrow batch: %w", err)
	}
	return nil
}

func (w *BatchWriter) Close() error {
	if w.writer == nil {
		return nil
	}
	err := w.writer.Close()
	w.writer = nil
	return err
}

// Decode reads an Arrow IPC stream into the engine's logical record shape.
// Integer values remain int64 instead of passing through float64, preserving
// identifiers above 2^53 exactly.
func Decode(r io.Reader) (Dataset, error) {
	reader, err := ipc.NewReader(r, ipc.WithAllocator(memory.DefaultAllocator))
	if err != nil {
		return Dataset{}, fmt.Errorf("open arrow ipc stream: %w", err)
	}
	defer reader.Release()
	columns := make([]string, 0, len(reader.Schema().Fields()))
	for _, field := range reader.Schema().Fields() {
		columns = append(columns, field.Name)
	}
	rows := make([]engine.Record, 0)
	for reader.Next() {
		record := reader.Record()
		for row := int64(0); row < record.NumRows(); row++ {
			out := make(engine.Record, len(columns))
			for column, values := range record.Columns() {
				value, err := valueAt(values, int(row))
				if err != nil {
					return Dataset{}, fmt.Errorf("column %q row %d: %w", columns[column], len(rows), err)
				}
				out[columns[column]] = value
			}
			rows = append(rows, out)
		}
	}
	if err := reader.Err(); err != nil && err != io.EOF {
		return Dataset{}, fmt.Errorf("read arrow ipc stream: %w", err)
	}
	return Dataset{Columns: columns, Rows: rows}, nil
}

func valueAt(values arrowgo.Array, row int) (any, error) {
	if values.IsNull(row) {
		return nil, nil
	}
	switch values := values.(type) {
	case *array.Boolean:
		return values.Value(row), nil
	case *array.String:
		return values.Value(row), nil
	case *array.LargeString:
		return values.Value(row), nil
	case *array.Int8:
		return int64(values.Value(row)), nil
	case *array.Int16:
		return int64(values.Value(row)), nil
	case *array.Int32:
		return int64(values.Value(row)), nil
	case *array.Int64:
		return values.Value(row), nil
	case *array.Uint8:
		return int64(values.Value(row)), nil
	case *array.Uint16:
		return int64(values.Value(row)), nil
	case *array.Uint32:
		return int64(values.Value(row)), nil
	case *array.Uint64:
		value := values.Value(row)
		if value > math.MaxInt64 {
			return nil, fmt.Errorf("uint64 value %d exceeds int64", value)
		}
		return int64(value), nil
	case *array.Float32:
		return float64(values.Value(row)), nil
	case *array.Float64:
		return values.Value(row), nil
	default:
		return nil, fmt.Errorf("unsupported arrow column type %s", values.DataType())
	}
}

// Encode writes records as an Arrow IPC stream. Column types are inferred
// from the first non-null value; callers that need a richer schema should
// construct Arrow directly and use Decode for the engine-facing side.
func Encode(w io.Writer, columns []string, rows []engine.Record) error {
	fields := make([]arrowgo.Field, len(columns))
	kinds := make([]string, len(columns))
	for i, column := range columns {
		kind := "string"
		for _, row := range rows {
			if value, ok := row[column]; ok && value != nil {
				kind = valueKind(value)
				break
			}
		}
		kinds[i] = kind
		fields[i] = arrowgo.Field{Name: column, Type: typeFor(kind), Nullable: true}
	}
	schema := arrowgo.NewSchema(fields, nil)
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer builder.Release()
	for _, row := range rows {
		for column, name := range columns {
			appendValue(builder.Field(column), kinds[column], row[name])
		}
	}
	record := builder.NewRecord()
	defer record.Release()
	writer := ipc.NewWriter(w, ipc.WithSchema(schema))
	if err := writer.Write(record); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write arrow record: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close arrow stream: %w", err)
	}
	return nil
}

func valueKind(value any) string {
	switch value.(type) {
	case bool:
		return "bool"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "int"
	case float32, float64:
		return "float"
	default:
		return "string"
	}
}

func typeFor(kind string) arrowgo.DataType {
	switch kind {
	case "bool":
		return arrowgo.FixedWidthTypes.Boolean
	case "int":
		return arrowgo.PrimitiveTypes.Int64
	case "float":
		return arrowgo.PrimitiveTypes.Float64
	default:
		return arrowgo.BinaryTypes.String
	}
}

func appendValue(builder array.Builder, kind string, value any) {
	if value == nil {
		builder.AppendNull()
		return
	}
	switch kind {
	case "bool":
		builder.(*array.BooleanBuilder).Append(value.(bool))
	case "int":
		builder.(*array.Int64Builder).Append(toInt64(value))
	case "float":
		builder.(*array.Float64Builder).Append(toFloat64(value))
	default:
		builder.(*array.StringBuilder).Append(fmt.Sprint(value))
	}
}

func toInt64(value any) int64 {
	switch value := value.(type) {
	case int:
		return int64(value)
	case int8:
		return int64(value)
	case int16:
		return int64(value)
	case int32:
		return int64(value)
	case int64:
		return value
	case uint:
		return int64(value)
	case uint8:
		return int64(value)
	case uint16:
		return int64(value)
	case uint32:
		return int64(value)
	case uint64:
		return int64(value)
	default:
		return 0
	}
}

func toFloat64(value any) float64 {
	switch value := value.(type) {
	case float32:
		return float64(value)
	case float64:
		return value
	default:
		return 0
	}
}
