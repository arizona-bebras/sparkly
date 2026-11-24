package domain

import (
	"time"

	"github.com/google/uuid"
)

// SymbolicatedBatch carries the normalized events and ownership from a symbolication result.
type SymbolicatedBatch struct {
	ID         uuid.UUID
	Identity   DeviceIdentity
	ReceivedAt time.Time
	Output     SymbolicatedOutput
}

type SymbolicatedOutput struct {
	Boots       []SymbolicatedBoot   `json:"boots"`
	Metrics     []SymbolicatedMetric `json:"metrics"`
	Logs        []SymbolicatedLog    `json:"logs"`
	Crashes     []SymbolicatedCrash  `json:"crashes"`
	Diagnostics []SymbolDiagnostic   `json:"diagnostics"`
}

type SymbolicatedEvent struct {
	Index       *uint64 `json:"event_index"`
	BootID      *string `json:"boot_id"`
	BuildID     *string `json:"build_id"`
	MonotonicUS *uint64 `json:"monotonic_us"`
}

type SymbolicatedBoot struct {
	SymbolicatedEvent
	ResetReason     string  `json:"reset_reason"`
	RawResetReason  *uint32 `json:"raw_reset_reason"`
	FirmwareVersion string  `json:"firmware_version"`
	AgentVersion    string  `json:"agent_version"`
}

type SymbolicatedMetric struct {
	SymbolicatedEvent
	MetricID     *uint64  `json:"metric_id"`
	Name         *string  `json:"name"`
	Kind         *string  `json:"kind"`
	Value        *float64 `json:"value"`
	CounterDelta *uint64  `json:"counter_delta"`
	DecodeStatus string   `json:"decode_status"`
}

type SymbolicatedFrame struct {
	PC       *uint64 `json:"pc"`
	Function *string `json:"function"`
	File     *string `json:"file"`
	Line     *uint32 `json:"line"`
}

type SymbolicatedLog struct {
	SymbolicatedEvent
	Level       *string             `json:"level"`
	Message     *string             `json:"message"`
	Module      *string             `json:"module"`
	Frames      []SymbolicatedFrame `json:"frames"`
	Truncated   bool                `json:"truncated"`
	TraceStatus string              `json:"trace_status"`
	DefmtRawHex *string             `json:"defmt_raw_hex"`
}

type SymbolicatedCrash struct {
	SymbolicatedEvent
	SP        *uint64             `json:"sp"`
	Stack     string              `json:"stack"`
	Registers map[string]uint64   `json:"registers"`
	Reason    string              `json:"reason"`
	Frames    []SymbolicatedFrame `json:"frames"`
}

type SymbolDiagnostic struct {
	EventIndex *uint64 `json:"event_index"`
	Message    string  `json:"message"`
	Unattached bool    `json:"-"`
}

func (out SymbolicatedOutput) DiagnosticsByEvent() map[uint64][]string {
	diagnostics := make(map[uint64][]string)
	for _, diagnostic := range out.Diagnostics {
		if diagnostic.EventIndex != nil {
			index := *diagnostic.EventIndex
			diagnostics[index] = append(diagnostics[index], diagnostic.Message)
		}
	}
	return diagnostics
}
