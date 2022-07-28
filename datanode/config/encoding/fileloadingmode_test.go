// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package encoding_test

import (
	"testing"

	"code.vegaprotocol.io/data-node/config/encoding"

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
