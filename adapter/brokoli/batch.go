package brokoli

import (
	"context"
	"fmt"
	"io"

	"github.com/Tnsor-Labs/actually-fine/contract"
	"github.com/Tnsor-Labs/actually-fine/engine"
	"github.com/Tnsor-Labs/actually-fine/transport"
)

type BatchSource interface {
	Next(context.Context) (transport.RecordBatch, error)
}

type BatchSink func(context.Context, transport.RecordBatch) error

type BatchRequest struct {
	Contract    contract.Contract
	Source      BatchSource
	Accepted    BatchSink
	Quarantined BatchSink
	Evidence    EvidenceSink
	Context     context.Context
}

// RunBatches evaluates records directly from a batch transport. The source
// and sinks own physical encoding; this layer only applies contract semantics
// and result evidence.
func RunBatches(request BatchRequest) (Summary, error) {
	if err := request.Contract.Validate(); err != nil {
		return Summary{}, fmt.Errorf("invalid contract: %w", err)
	}
	if request.Source == nil {
		return Summary{}, fmt.Errorf("batch source is required")
	}
	ctx := request.Context
	if ctx == nil {
		ctx = context.Background()
	}

	var summary Summary
	checker := engine.NewStreamChecker(request.Contract)
	for {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		batch, err := request.Source.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return summary, fmt.Errorf("read batch: %w", err)
		}
		accepted := make([]map[string]any, 0, len(batch.Records))
		quarantined := make([]map[string]any, 0, len(batch.Records))
		for _, raw := range batch.Records {
			line := summary.Total + 1
			summary.Total++
			record := engine.Record(raw)
			violations := checker.Check(record)
			hasBreach, shouldQuarantine, shouldHalt := false, false, false
			for _, violation := range violations {
				if violation.Action == "warn" {
					summary.Warnings++
				} else {
					hasBreach = true
				}
				shouldQuarantine = shouldQuarantine || violation.Action == "quarantine"
				shouldHalt = shouldHalt || violation.Action == "halt"
				if err := publish(request.Evidence, makeEvent(request.Contract, record, line, violation)); err != nil {
					return summary, err
				}
			}
			switch {
			case !hasBreach:
				summary.Cleared++
				accepted = append(accepted, raw)
			case shouldQuarantine:
				summary.Quarantined++
				quarantined = append(quarantined, raw)
			default:
				summary.Breached++
			}
			if shouldHalt {
				summary.Halted = true
				break
			}
		}
		if len(accepted) > 0 && request.Accepted != nil {
			batch.Records = accepted
			if err := request.Accepted(ctx, batch); err != nil {
				return summary, fmt.Errorf("write accepted batch: %w", err)
			}
		}
		if len(quarantined) > 0 && request.Quarantined != nil {
			batch.Records = quarantined
			if err := request.Quarantined(ctx, batch); err != nil {
				return summary, fmt.Errorf("write quarantined batch: %w", err)
			}
		}
		if summary.Halted {
			break
		}
	}
	if !summary.Halted {
		for _, violation := range checker.Finalize() {
			if violation.Action == "warn" {
				summary.Warnings++
			} else {
				summary.Breached++
			}
			if err := publish(request.Evidence, makeEvent(request.Contract, engine.Record{}, max(1, summary.Total), violation)); err != nil {
				return summary, err
			}
		}
	}
	return summary, nil
}
