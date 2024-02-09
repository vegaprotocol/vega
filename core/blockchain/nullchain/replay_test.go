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
