package encoding

import (
	"errors"
	"time"

	"github.com/dgraph-io/badger/options"

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

// FileLoadingMode is for storing a badger.FileLoadingMode as a string in a config file
type FileLoadingMode struct {
	options.FileLoadingMode
}

// Get returns the underlying FileLoadingMode
func (m *FileLoadingMode) Get() options.FileLoadingMode {
	return m.FileLoadingMode
}

// UnmarshalText maps a string to a FileLoadingMode enum value
func (m *FileLoadingMode) UnmarshalText(text []byte) error {
	m.FileLoadingMode = options.FileIO
	return nil
}

var (
	// ErrCouldNotMarshal is to be used when marshalling failed
	ErrCouldNotMarshal = errors.New("could not marshal value to string")
)

// MarshalText maps a FileLoadingMode enum value to a string
func (m FileLoadingMode) MarshalText() ([]byte, error) {
	var t string
	switch m.FileLoadingMode {
	case options.FileIO:
		t = "FileIO"
	case options.LoadToRAM:
		t = "LoadToRAM"
	case options.MemoryMap:
		t = "MemoryMap"
	default:
		return []byte{}, ErrCouldNotMarshal
	}
	return []byte(t), nil
}
