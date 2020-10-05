package encoding

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

// Duration is a wrapper over an actual duration so we can represent
// them as string in the toml configuration
type Duration struct {
	time.Duration
}

// Get returns the stored duration
func (d *Duration) Get() time.Duration {
	return d.Duration
}

// UnmarshalText unmarshal a duration from bytes
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

func (d *Duration) UnmarshalFlag(s string) error {
	return d.UnmarshalText([]byte(s))
}

// MarshalText marshal a duraton into bytes
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// LogLevel is wrapper over the actual log level
// so they can be specified as strings in the toml configuration
type LogLevel struct {
	logging.Level
}

// Get return the store value
func (l *LogLevel) Get() logging.Level {
	return l.Level
}

// UnmarshalText unmarshal a loglevel from bytes
func (l *LogLevel) UnmarshalText(text []byte) error {
	var err error
	l.Level, err = logging.ParseLevel(string(text))
	return err
}

func (l *LogLevel) UnmarshalFlag(s string) error {
	return l.UnmarshalText([]byte(s))
}

// MarshalText marshal a loglevel into bytes
func (l LogLevel) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

type Bool bool

func (b *Bool) UnmarshalFlag(s string) error {
	if s == "true" {
		*b = true
	} else if s == "false" {
		*b = false
	} else {
		return fmt.Errorf("only `true' and `false' are valid values, not `%s'", s)
	}
	return nil
}
