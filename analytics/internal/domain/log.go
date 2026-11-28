package domain

import (
	"time"

	"github.com/google/uuid"
)

type LogLevel int

const (
	UNSPECIFIED LogLevel = iota
	TRACE
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

func LogLevelFromString(s string) LogLevel {
	switch s {
	case "TRACE", "trace":
		return TRACE
	case "DEBUG", "debug":
		return DEBUG
	case "INFO", "info":
		return INFO
	case "WARN", "warn":
		return INFO
	case "ERROR", "error":
		return ERROR
	case "FATAL", "fatal":
		return FATAL
	}

	return UNSPECIFIED
}

type StackFrame struct {
	File     *string
	Line     uint32
	Function *string
	PC       uint64
}

type Log struct {
	ParkID      uuid.UUID
	DeviceID    uuid.UUID
	BootID      *uuid.UUID
	BuildID     *uuid.UUID
	LogLevel    LogLevel
	Message     string
	Timestamp   time.Time
	StackFrames []StackFrame
	Attributes  map[string]string
}
