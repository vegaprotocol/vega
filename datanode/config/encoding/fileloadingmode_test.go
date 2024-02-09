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

package encoding_test

import (
	"testing"

	"code.vegaprotocol.io/vega/datanode/config/encoding"

	"github.com/dgraph-io/badger/v2/options"
	"github.com/stretchr/testify/assert"
)

func TestFileLoadingMode(t *testing.T) {
	var flm encoding.FileLoadingMode
	var flmbytes []byte
	var err error

	flmstrs := []string{"FileIO", "MemoryMap", "LoadToRAM"}
	flms := []options.FileLoadingMode{options.FileIO, options.MemoryMap, options.LoadToRAM}
	for i, flmstr := range flmstrs {
		err = flm.UnmarshalText([]byte(flmstr))
		assert.NoError(t, err)
		assert.Equal(t, flms[i], flm.Get())

		flmbytes, err = flm.MarshalText()
		assert.NoError(t, err)
		assert.Equal(t, flmstr, string(flmbytes))
	}

	err = flm.UnmarshalText([]byte("this is not a fileloadingmode"))
	assert.Equal(t, encoding.ErrCouldNotUnmarshalFLM, err)
}
