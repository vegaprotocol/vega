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

package pow

import (
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/pow/mocks"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestConversions(t *testing.T) {
	p := &types.PayloadProofOfWork{
		BlockHeight: []uint64{100, 101, 102},
		BlockHash:   []string{"94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", "DC911C0EA95545441F3E1182DD25D973764395A7E75CBDBC086F1C6F7075AED6", "2E4F2967AA904F9A952BB4813EC6BBB3730B9FFFEC44106B89F0A1958547733C"},
		HeightToTx:  map[uint64][]string{100: {"1", "2"}, 101: {"3"}},
		HeightToTid: map[uint64][]string{100: {"100", "200"}, 101: {"300"}},
	}

	pp := p.IntoProto()
	require.Equal(t, 3, len(pp.ProofOfWork.BlockHeight))
	require.Equal(t, 3, len(pp.ProofOfWork.BlockHash))
	for i, v := range p.BlockHeight {
		require.Equal(t, v, pp.ProofOfWork.BlockHeight[i])
	}
	for i, v := range p.BlockHash {
		require.Equal(t, v, pp.ProofOfWork.BlockHash[i])
	}
	require.Equal(t, 2, len(pp.ProofOfWork.TxAtHeight))
	require.Equal(t, 2, len(pp.ProofOfWork.TidAtHeight))
	require.Equal(t, uint64(100), pp.ProofOfWork.TxAtHeight[0].Height)
	require.Equal(t, uint64(101), pp.ProofOfWork.TxAtHeight[1].Height)
	require.Equal(t, uint64(100), pp.ProofOfWork.TidAtHeight[0].Height)
	require.Equal(t, uint64(101), pp.ProofOfWork.TidAtHeight[1].Height)
	require.Equal(t, 2, len(pp.ProofOfWork.TxAtHeight[0].Transactions))
	require.Equal(t, 2, len(pp.ProofOfWork.TidAtHeight[0].Transactions))
	require.Equal(t, "1", pp.ProofOfWork.TxAtHeight[0].Transactions[0])
	require.Equal(t, "2", pp.ProofOfWork.TxAtHeight[0].Transactions[1])
	require.Equal(t, "3", pp.ProofOfWork.TxAtHeight[1].Transactions[0])
	require.Equal(t, "100", pp.ProofOfWork.TidAtHeight[0].Transactions[0])
	require.Equal(t, "200", pp.ProofOfWork.TidAtHeight[0].Transactions[1])
	require.Equal(t, "300", pp.ProofOfWork.TidAtHeight[1].Transactions[0])

	ppp := types.PayloadProofOfWorkFromProto(pp)

	require.Equal(t, 3, len(ppp.BlockHeight))
	require.Equal(t, 3, len(ppp.BlockHash))
	for i, v := range ppp.BlockHeight {
		require.Equal(t, v, pp.ProofOfWork.BlockHeight[i])
	}
	for i, v := range ppp.BlockHash {
		require.Equal(t, v, pp.ProofOfWork.BlockHash[i])
	}
	require.Equal(t, 2, len(ppp.HeightToTx))
	require.Equal(t, 2, len(ppp.HeightToTid))
	require.Equal(t, 2, len(ppp.HeightToTx[100]))
	require.Equal(t, 2, len(ppp.HeightToTid[100]))
	require.Equal(t, "1", ppp.HeightToTx[100][0])
	require.Equal(t, "2", ppp.HeightToTx[100][1])
	require.Equal(t, "100", ppp.HeightToTid[100][0])
	require.Equal(t, "200", ppp.HeightToTid[100][1])
	require.Equal(t, 1, len(ppp.HeightToTx[101]))
	require.Equal(t, 1, len(ppp.HeightToTid[101]))
	require.Equal(t, "3", ppp.HeightToTx[101][0])
	require.Equal(t, "300", ppp.HeightToTid[101][0])
}

func TestSnapshot(t *testing.T) {
	ts := mocks.NewMockTimeService(gomock.NewController(t))
	ts.EXPECT().GetTimeNow().AnyTimes().Return(time.Now())
	e := New(logging.NewTestLogger(), NewDefaultConfig())
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(100))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))

	e.BeginBlock(100, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", []abci.Tx{})

	// add a new set of configuration which becomes active at block 100
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(200))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(25))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(2))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(0))

	party := crypto.RandomHash()

	txs := []abci.Tx{
		&testTx{txID: "1", party: party, blockHeight: 100, powTxID: "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F", powNonce: 424517},
		&testTx{txID: "2", party: party, blockHeight: 100, powTxID: "5B0E1EB96CCAC120E6D824A5F4C4007EABC59573B861BD84B1EF09DFB376DC84", powNonce: 4031737},
		&testTx{txID: "3", party: party, blockHeight: 100, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336},
	}

	e.BeginBlock(101, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F", txs)
	e.BeginBlock(102, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F", []abci.Tx{})

	key := (&types.PayloadProofOfWork{}).Key()
	state1, _, err := e.GetState(key)
	require.NoError(t, err)

	eLoaded := New(logging.NewTestLogger(), NewDefaultConfig())
	eLoaded.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(1))
	eLoaded.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	eLoaded.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	eLoaded.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state1, &pl))
	eLoaded.LoadState(context.Background(), types.PayloadFromProto(&pl))

	state2, _, err := eLoaded.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state1, state2))

	require.Equal(t, 2, len(eLoaded.activeParams))
}

func TestSnapshotViaEngine(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)
	ts := mocks.NewMockTimeService(gomock.NewController(t))
	ts.EXPECT().GetTimeNow().AnyTimes().Return(time.Now())
	powEngine1 := New(logging.NewTestLogger(), NewDefaultConfig())
	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snapshot.DefaultConfig()
	vegaPath := paths.New(t.TempDir())
	snapshotEngine1, err := snapshot.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	snapshotEngine1CloseFn := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer snapshotEngine1CloseFn()

	snapshotEngine1.AddProviders(powEngine1)

	require.NoError(t, snapshotEngine1.Start(ctx))
	require.NoError(t, powEngine1.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(2)))
	require.NoError(t, powEngine1.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20)))
	require.NoError(t, powEngine1.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3))
	require.NoError(t, powEngine1.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1)))
	require.NoError(t, powEngine1.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1)))

	powEngine1.BeginBlock(99, "377EEAC9847D751A4FAFD3F2896E99C1A03363EBDA3036C33940CFE578E196D1", []abci.Tx{})
	powEngine1.BeginBlock(100, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", []abci.Tx{})

	party := crypto.RandomHash()

	txs := []abci.Tx{
		&testTx{txID: "1", party: party, blockHeight: 100, powTxID: "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F", powNonce: 424517},
		&testTx{txID: "2", party: party, blockHeight: 100, powTxID: "5B0E1EB96CCAC120E6D824A5F4C4007EABC59573B861BD84B1EF09DFB376DC84", powNonce: 4031737},
		&testTx{txID: "3", party: party, blockHeight: 100, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336},
		// add another transaction from the same party with reduced difficulty but from another block
		&testTx{txID: "4", party: party, blockHeight: 99, powTxID: "4633a4d29f543cdd9afe7555c352179063d1ead0c778d246fabfc4c6f8adf031", powNonce: 2646611},
	}

	powEngine1.BeginBlock(101, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F", txs)
	powEngine1.BeginBlock(102, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F", []abci.Tx{})

	require.NoError(t, powEngine1.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(2)))
	require.NoError(t, powEngine1.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(25)))
	require.NoError(t, powEngine1.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3))
	require.NoError(t, powEngine1.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(5)))
	require.NoError(t, powEngine1.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(0)))

	ctx = vgcontext.WithTraceID(vgcontext.WithBlockHeight(context.Background(), 102), "0xDEADBEEF")
	ctx = vgcontext.WithChainID(ctx, "chainid")
	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)
	state1 := map[string][]byte{}
	for _, key := range powEngine1.Keys() {
		state, additionalProvider, err := powEngine1.GetState(key)
		require.NoError(t, err)
		require.Empty(t, additionalProvider)
		state1[key] = state
	}

	snapshotEngine1CloseFn()

	tsLoaded := mocks.NewMockTimeService(gomock.NewController(t))
	tsLoaded.EXPECT().GetTimeNow().AnyTimes().Return(time.Now())
	powEngine2 := New(logging.NewTestLogger(), NewDefaultConfig())
	timeServiceLoaded := stubs.NewTimeStub()
	timeServiceLoaded.SetTime(now)
	snapshotEngine2, err := snapshot.NewEngine(vegaPath, config, log, timeServiceLoaded, statsData.Blockchain)
	require.NoError(t, err)
	defer snapshotEngine2.Close()

	snapshotEngine2.AddProviders(powEngine2)

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	state2 := map[string][]byte{}
	for _, key := range powEngine2.Keys() {
		state, additionalProvider, err := powEngine2.GetState(key)
		require.NoError(t, err)
		require.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		require.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}
