// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
