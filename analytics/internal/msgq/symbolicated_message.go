package msgq

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"sparkly/analytics/internal/domain"
)

type symbolicationResultMessage struct {
	SchemaVersion int                        `json:"schema_version"`
	ResultVersion int                        `json:"result_version"`
	BatchID       uuid.UUID                  `json:"batch_id"`
	DeviceID      uuid.UUID                  `json:"device_id"`
	ParkID        uuid.UUID                  `json:"park_id"`
	ReceivedAt    time.Time                  `json:"received_at"`
	Result        *symbolicationResultOutput `json:"result"`
}

type symbolicationResultOutput struct {
	domain.SymbolicatedOutput
	Logs []symbolicationResultLog `json:"logs"`
}

type symbolicationResultLog struct {
	domain.SymbolicatedLog
	Frames []symbolicationResultFrame `json:"frames"`
	Target *string                    `json:"target"`
	Source *struct {
		Module *string `json:"module"`
	} `json:"source"`
}

type symbolicationResultFrame struct {
	domain.SymbolicatedFrame
	Module *string `json:"module"`
}

func (result *symbolicationResultOutput) normalize(version int) domain.SymbolicatedOutput {
	out := result.SymbolicatedOutput
	if result.Logs != nil {
		out.Logs = make([]domain.SymbolicatedLog, 0, len(result.Logs))
		for _, entry := range result.Logs {
			row := entry.SymbolicatedLog
			frames := entry.Frames
			switch {
			case version == 1:
				row.Module = entry.Target
			case version < 4 && len(frames) > 0 && frames[0].PC == nil && frames[0].Function == nil:
				// Versions 2 and 3 carried the log site in an artificial first frame.
				row.Module = frames[0].Module
				frames = frames[1:]
			case version == 4 && entry.Source != nil:
				row.Module = entry.Source.Module
			}
			if frames != nil {
				row.Frames = make([]domain.SymbolicatedFrame, 0, len(frames))
				for _, frame := range frames {
					row.Frames = append(row.Frames, frame.SymbolicatedFrame)
				}
			}
			out.Logs = append(out.Logs, row)
		}
	}
	return out
}

func decodeSymbolicatedBatch(msg amqp.Delivery) (domain.SymbolicatedBatch, error) {
	var batch domain.SymbolicatedBatch
	if msg.ContentType != "application/json" {
		return batch, fmt.Errorf("expected application/json")
	}
	var envelope symbolicationResultMessage
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		return batch, fmt.Errorf("decode result envelope: %w", err)
	}
	if envelope.SchemaVersion != 1 || envelope.ResultVersion < 1 || envelope.ResultVersion > 5 {
		return batch, fmt.Errorf("unsupported symbolication schema/result version %d/%d", envelope.SchemaVersion, envelope.ResultVersion)
	}
	if envelope.BatchID == uuid.Nil {
		return batch, fmt.Errorf("batch ID is required")
	}
	id := envelope.BatchID.String()
	if (msg.MessageId != "" && msg.MessageId != id) || (msg.CorrelationId != "" && msg.CorrelationId != id) {
		return batch, fmt.Errorf("AMQP IDs must match the batch ID")
	}
	identity, err := domain.NewDeviceIdentity(envelope.DeviceID, envelope.ParkID)
	if err != nil {
		return batch, err
	}
	// Keep the original receipt time at nanosecond precision in ClickHouse.
	if envelope.ReceivedAt.Before(time.Unix(0, 0)) || envelope.ReceivedAt.After(time.Unix(0, math.MaxInt64)) {
		return batch, fmt.Errorf("receipt time outside DateTime64(9) range")
	}
	if envelope.Result == nil {
		return batch, fmt.Errorf("result is required")
	}
	out := envelope.Result.normalize(envelope.ResultVersion)
	if err := validateSymbolicatedOutput(out); err != nil {
		return batch, err
	}
	return domain.SymbolicatedBatch{ID: envelope.BatchID, Identity: identity, ReceivedAt: envelope.ReceivedAt.UTC(), Output: out}, nil
}

func validateSymbolicatedOutput(out domain.SymbolicatedOutput) error {
	if out.Boots == nil || out.Metrics == nil || out.Logs == nil || out.Crashes == nil || out.Diagnostics == nil {
		return fmt.Errorf("result must contain boots, metrics, logs, crashes and diagnostics arrays")
	}
	seen := make(map[uint64]struct{})
	validateEvent := func(event domain.SymbolicatedEvent) error {
		if event.Index == nil || event.MonotonicUS == nil {
			return fmt.Errorf("event_index and monotonic_us are required")
		}
		index := *event.Index
		if _, exists := seen[index]; exists {
			return fmt.Errorf("duplicate event index %d", index)
		}
		seen[index] = struct{}{}
		return nil
	}
	for _, event := range out.Boots {
		if err := validateEvent(event.SymbolicatedEvent); err != nil {
			return err
		}
	}
	for _, event := range out.Metrics {
		if err := validateEvent(event.SymbolicatedEvent); err != nil {
			return err
		}
		if event.MetricID == nil || event.DecodeStatus == "" || (event.Value != nil && event.CounterDelta != nil) {
			return fmt.Errorf("invalid metric at event index %d", *event.Index)
		}
	}
	for _, event := range out.Logs {
		if err := validateEvent(event.SymbolicatedEvent); err != nil {
			return err
		}
		if event.Frames == nil {
			return fmt.Errorf("frames array is required at event index %d", *event.Index)
		}
		for _, frame := range event.Frames {
			if frame.PC == nil {
				return fmt.Errorf("frame PC is required at event index %d", *event.Index)
			}
		}
	}
	for _, event := range out.Crashes {
		if err := validateEvent(event.SymbolicatedEvent); err != nil {
			return err
		}
		if event.Registers == nil || event.Frames == nil {
			return fmt.Errorf("invalid crash at event index %d", *event.Index)
		}
		for _, frame := range event.Frames {
			if frame.PC == nil {
				return fmt.Errorf("frame PC is required at event index %d", *event.Index)
			}
		}
	}
	for i := range out.Diagnostics {
		diagnostic := &out.Diagnostics[i]
		if diagnostic.Message == "" {
			return fmt.Errorf("diagnostic message is required")
		}
		if diagnostic.EventIndex != nil {
			_, attached := seen[*diagnostic.EventIndex]
			diagnostic.Unattached = !attached
		}
	}
	return nil
}
