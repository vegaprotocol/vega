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

package storage_test

import (
	"path/filepath"
	"testing"

	vgtesting "code.vegaprotocol.io/data-node/libs/testing"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testChainID = "purple panda"

func TestStorage_SaveLoadChainInfo(t *testing.T) {
	config, err := storage.NewTestConfig()
	require.NoError(t, err)

	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()

	st, err := storage.InitialiseStorage(vegaPaths)
	defer st.Purge()
	require.NoError(t, err)

	chainInfo, err := storage.NewChainInfo(logging.NewTestLogger(),
		st.ChainInfoHome,
		config,
		func() {})
	assert.NoError(t, err)

	chainInfo.SetChainID(testChainID)
	require.FileExists(t, filepath.Join(st.ChainInfoHome, "info.json"))

	retrievedChainId, err := chainInfo.GetChainID()
	assert.NoError(t, err)
	assert.Equal(t, retrievedChainId, testChainID)
}
