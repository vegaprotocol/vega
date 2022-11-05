// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nullchain_test

import (
	"os"
	"path"
	"testing"

	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/core/blockchain/nullchain"
	"code.vegaprotocol.io/vega/core/blockchain/nullchain/mocks"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestNullchainReplayer(t *testing.T) {
	t.Run("test no file provided", testReplayerNoFile)
	t.Run("test truncate if record but not replay", testTruncateIfRecordButNoReplay)
}

func testReplayerNoFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	app := mocks.NewMockApplicationService(ctrl)
	defer ctrl.Finish()
	r, err := nullchain.NewNullChainReplayer(app, blockchain.ReplayConfig{}, logging.NewTestLogger())
	require.ErrorIs(t, err, nullchain.ErrReplayFileIsRequired)
	require.Nil(t, r)
}

func testTruncateIfRecordButNoReplay(t *testing.T) {
	ctrl := gomock.NewController(t)
	app := mocks.NewMockApplicationService(ctrl)
	defer ctrl.Finish()

	// write some nonsense into the replay file as if we've recorded something
	rplFile := path.Join(t.TempDir(), "rfile")
	f, err := os.Create(rplFile)
	require.NoError(t, err)

	f.WriteString(vgrand.RandomStr(5))
	f.Close()

	r, err := nullchain.NewNullChainReplayer(app, blockchain.ReplayConfig{Record: true, Replay: false, ReplayFile: rplFile}, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, r)
	defer r.Stop()

	// check that the file is now empty
	info, err := os.Stat(rplFile)
	require.NoError(t, err)
	require.Equal(t, int64(0), info.Size())
}
