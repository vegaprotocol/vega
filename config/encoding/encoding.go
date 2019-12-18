package encoding

import (
	"fmt"
	"reflect"
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

// MarshalText marshal a duraton into bytes
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func DurationDecodeHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t == reflect.TypeOf(Duration{}) {
		val, ok := data.(string)
		if !ok {
			return data, fmt.Errorf("expected a string to unwrap Duration")
		}
		var (
			dur Duration
			err error
		)
		dur.Duration, err = time.ParseDuration(val)
		if err != nil {
			return data, err
		}
		return dur, nil
	}
	return data, nil
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

// MarshalText marshal a loglevel into bytes
func (l LogLevel) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

func LogLevelDecodeHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t == reflect.TypeOf(LogLevel{}) {
		val, ok := data.(string)
		if !ok {
			return data, fmt.Errorf("expected a string to unwrap LogLevel")
		}

		var (
			lvl LogLevel
			err error
		)
		lvl.Level, err = logging.ParseLevel(val)
		if err != nil {
			return data, err
		}
		return lvl, nil
	}

	return data, nil
}
