package arrow

import (
	"context"
	"fmt"
	"io"

	"github.com/Tnsor-Labs/actually-fine/adapter/brokoli"
	"github.com/Tnsor-Labs/actually-fine/contract"
	"github.com/Tnsor-Labs/actually-fine/transport"
)

type GateRequest struct {
	Contract    contract.Contract
	Input       io.Reader
	Accepted    io.Writer
	Quarantined io.Writer
	Evidence    brokoli.EvidenceSink
	Context     context.Context
}

// RunGate connects Arrow IPC directly to the batch gate. Records are decoded
// into logical batches, evaluated, and encoded back to Arrow without an
// intermediate NDJSON representation.
func RunGate(request GateRequest) (brokoli.Summary, error) {
	reader, err := NewBatchReader(request.Input)
	if err != nil {
		return brokoli.Summary{}, err
	}
	defer reader.Close()

	var accepted, quarantined *BatchWriter
	batchRequest := brokoli.BatchRequest{
		Contract: request.Contract,
		Source:   reader,
		Evidence: request.Evidence,
		Context:  request.Context,
	}
	if request.Accepted != nil {
		accepted = NewBatchWriter(request.Accepted)
		batchRequest.Accepted = func(ctx context.Context, batch transport.RecordBatch) error {
			return accepted.Write(ctx, batch)
		}
	}
	if request.Quarantined != nil {
		quarantined = NewBatchWriter(request.Quarantined)
		batchRequest.Quarantined = func(ctx context.Context, batch transport.RecordBatch) error {
			return quarantined.Write(ctx, batch)
		}
	}

	summary, runErr := brokoli.RunBatches(batchRequest)
	if err := closeWriters(accepted, quarantined); runErr == nil && err != nil {
		runErr = err
	}
	return summary, runErr
}

func closeWriters(writers ...*BatchWriter) error {
	for _, writer := range writers {
		if writer == nil {
			continue
		}
		if err := writer.Close(); err != nil {
			return fmt.Errorf("close arrow output: %w", err)
		}
	}
	return nil
}
