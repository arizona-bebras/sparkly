package repos

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"sparkly/analytics/internal/domain"
)

type SymbolicatedEventsRepo struct {
	db driver.Conn
}

func NewSymbolicatedEventsRepo(db driver.Conn) *SymbolicatedEventsRepo {
	return &SymbolicatedEventsRepo{db: db}
}

// StoreSymbolicatedBatch uses a separate native ClickHouse insert per table.
// Rows written before a failed insert retain the same keys on delivery retry.
func (r *SymbolicatedEventsRepo) StoreSymbolicatedBatch(ctx context.Context, batch domain.SymbolicatedBatch) error {
	out := batch.Output
	diagnostics := out.DiagnosticsByEvent()
	common := func(event domain.SymbolicatedEvent) []any {
		return []any{batch.Identity.ParkID().String(), batch.Identity.DeviceID().String(), batch.ID.String(),
			*event.Index, batch.ReceivedAt, event.BootID, event.BuildID, *event.MonotonicUS, diagnostics[*event.Index]}
	}

	logs := make([][]any, 0, len(out.Logs))
	for _, event := range out.Logs {
		logs = append(logs, append(common(event.SymbolicatedEvent), event.Level, event.Message,
			event.Module, frameTuples(event.Frames), event.Truncated, event.TraceStatus, event.DefmtRawHex))
	}
	if err := r.insertEvents(ctx, insertSymbolicatedLogs, logs); err != nil {
		return fmt.Errorf("insert logs: %w", err)
	}

	metrics := make([][]any, 0, len(out.Metrics))
	for _, event := range out.Metrics {
		metrics = append(metrics, append(common(event.SymbolicatedEvent), *event.MetricID, event.Name,
			event.Kind, event.Value, event.CounterDelta, event.DecodeStatus))
	}
	if err := r.insertEvents(ctx, insertSymbolicatedMetrics, metrics); err != nil {
		return fmt.Errorf("insert metrics: %w", err)
	}

	boots := make([][]any, 0, len(out.Boots))
	for _, event := range out.Boots {
		boots = append(boots, append(common(event.SymbolicatedEvent), event.ResetReason, event.RawResetReason,
			event.FirmwareVersion, event.AgentVersion))
	}
	if err := r.insertEvents(ctx, insertSymbolicatedBoots, boots); err != nil {
		return fmt.Errorf("insert boots: %w", err)
	}

	crashes := make([][]any, 0, len(out.Crashes))
	for _, event := range out.Crashes {
		crashes = append(crashes, append(common(event.SymbolicatedEvent), frameTuples(event.Frames),
			event.SP, event.Reason, event.Stack, event.Registers))
	}
	if err := r.insertEvents(ctx, insertSymbolicatedCrashes, crashes); err != nil {
		return fmt.Errorf("insert crashes: %w", err)
	}

	return nil
}

const symbolicatedEventColumns = "park_id, device_id, batch_id, event_index, received_at, boot_id, build_id, monotonic_us, diagnostics"

const insertSymbolicatedLogs = "INSERT INTO logs (" + symbolicatedEventColumns + `,
	level, message, module, frames, truncated, trace_status, defmt_raw_hex)`

const insertSymbolicatedMetrics = "INSERT INTO metrics (" + symbolicatedEventColumns + `,
	metric_id, name, kind, value, counter_delta, decode_status)`

const insertSymbolicatedBoots = "INSERT INTO boots (" + symbolicatedEventColumns + `,
	reset_reason, raw_reset_reason, firmware_version, agent_version)`

const insertSymbolicatedCrashes = "INSERT INTO crashes (" + symbolicatedEventColumns + `,
	frames, sp, reason, stack_hex, registers)`

func frameTuples(frames []domain.SymbolicatedFrame) [][]any {
	tuples := make([][]any, 0, len(frames))
	for _, frame := range frames {
		tuples = append(tuples, []any{*frame.PC, frame.Function, frame.File, frame.Line})
	}
	return tuples
}

func (r *SymbolicatedEventsRepo) insertEvents(ctx context.Context, query string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := r.db.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}
	defer batch.Abort()
	for _, row := range rows {
		if err := batch.Append(row...); err != nil {
			return err
		}
	}
	return batch.Send()
}
