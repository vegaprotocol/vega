package encoding

import (
	"errors"

	"github.com/dgraph-io/badger/options"
)

// FileLoadingMode is for storing a badger.FileLoadingMode as a string in a config file
type FileLoadingMode struct {
	options.FileLoadingMode
}

// Get returns the underlying FileLoadingMode
func (m *FileLoadingMode) Get() options.FileLoadingMode {
	return m.FileLoadingMode
}

var (
	// ErrCouldNotMarshal is to be used when marshalling failed
	ErrCouldNotMarshal = errors.New("could not marshal value to string")

	// ErrCouldNotUnmarshal is to be used when unmarshalling failed
	ErrCouldNotUnmarshal = errors.New("could not unmarshal value from string")
)

// UnmarshalText maps a string to a FileLoadingMode enum value
func (m *FileLoadingMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "FileIO":
		m.FileLoadingMode = options.FileIO
	case "LoadToRAM":
		m.FileLoadingMode = options.LoadToRAM
	case "MemoryMap":
		m.FileLoadingMode = options.MemoryMap
	default:
		return ErrCouldNotUnmarshal
	}
	return nil
}

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
