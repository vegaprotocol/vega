package encoding

import (
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
)

type Duration struct {
	time.Duration
}

func (d *Duration) Get() time.Duration {
	return d.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

type LogLevel struct {
	logging.Level
}

func (l *LogLevel) Get() logging.Level {
	return l.Level
}

func (l *LogLevel) UnmarshalText(text []byte) error {
	var err error
	l.Level, err = logging.ParseLevel(string(text))
	return err
}

func (l LogLevel) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}
