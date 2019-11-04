package scenariorunner

import (
	"time"
)

type Config struct {
	ProtocolTime                time.Time
	AdvanceTimeAfterInstruction bool
	AdvanceDuration             time.Duration
	OmitUnsupportedInstructions bool
	OmitInvalidInstructions     bool
}

func NewDefaultConfig() Config {
	return Config{
		ProtocolTime:                time.Date(2019, 1, 2, 8, 0, 0, 0, time.UTC),
		AdvanceTimeAfterInstruction: true,
		AdvanceDuration:             time.Nanosecond,
		OmitUnsupportedInstructions: true,
		OmitInvalidInstructions:     true,
	}
}
