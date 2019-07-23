package encoding_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/config/encoding"

	"github.com/dgraph-io/badger/options"
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
