// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package encoding

import (
	"errors"

	"github.com/dgraph-io/badger/v2/options"
)

var (
	// ErrCouldNotMarshalFLM is to be used when marshalling failed.
	ErrCouldNotMarshalFLM = errors.New("could not marshal FileLoadingMode value to string")
	// ErrCouldNotUnmarshalFLM is to be used when unmarshalling failed.
	ErrCouldNotUnmarshalFLM = errors.New("could not unmarshal FileLoadingMode value from string")
)

// FileLoadingMode is for storing a badger.FileLoadingMode as a string in a config file.
type FileLoadingMode struct {
	options.FileLoadingMode
}

// Get returns the underlying FileLoadingMode.
func (m *FileLoadingMode) Get() options.FileLoadingMode {
	return m.FileLoadingMode
}

// UnmarshalText maps a string to a FileLoadingMode enum value.
func (m *FileLoadingMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "FileIO":
		m.FileLoadingMode = options.FileIO
	case "LoadToRAM":
		m.FileLoadingMode = options.LoadToRAM
	case "MemoryMap":
		m.FileLoadingMode = options.MemoryMap
	default:
		return ErrCouldNotUnmarshalFLM
	}
	return nil
}

// MarshalText maps a FileLoadingMode enum value to a string.
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
		return []byte{}, ErrCouldNotMarshalFLM
	}
	return []byte(t), nil
}
