package pow

import (
	"bytes"
	"context"
	"testing"

	"code.vegaprotocol.io/protos/vega"
	snapshotpb "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/shared/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	gtypes "code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestConversions(t *testing.T) {
	p := &types.PayloadProofOfWork{
		BlockHeight:   []uint64{100, 101, 102},
		BlockHash:     []string{"94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", "DC911C0EA95545441F3E1182DD25D973764395A7E75CBDBC086F1C6F7075AED6", "2E4F2967AA904F9A952BB4813EC6BBB3730B9FFFEC44106B89F0A1958547733C"},
		SeenTx:        map[string]struct{}{"1": {}, "2": {}, "3": {}},
		HeightToTx:    map[uint64][]string{100: {"1", "2"}, 101: {"3"}},
		SeenTid:       map[string]struct{}{"100": {}, "200": {}, "300": {}},
		HeightToTid:   map[uint64][]string{100: {"100", "200"}, 101: {"300"}},
		BannedParties: map[string]uint64{"party1": 105, "party2": 104},
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
	require.Equal(t, "1", pp.ProofOfWork.SeenTx[0])
	require.Equal(t, "2", pp.ProofOfWork.SeenTx[1])
	require.Equal(t, "3", pp.ProofOfWork.SeenTx[2])
	require.Equal(t, "100", pp.ProofOfWork.SeenTid[0])
	require.Equal(t, "200", pp.ProofOfWork.SeenTid[1])
	require.Equal(t, "300", pp.ProofOfWork.SeenTid[2])
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
	require.Equal(t, "party1", pp.ProofOfWork.Banned[0].Party)
	require.Equal(t, uint64(105), pp.ProofOfWork.Banned[0].UntilEpoch)
	require.Equal(t, "party2", pp.ProofOfWork.Banned[1].Party)
	require.Equal(t, uint64(104), pp.ProofOfWork.Banned[1].UntilEpoch)

	ppp := types.PayloadProofOfWorkFromProto(pp)

	require.Equal(t, 3, len(ppp.BlockHeight))
	require.Equal(t, 3, len(ppp.BlockHash))
	for i, v := range ppp.BlockHeight {
		require.Equal(t, v, pp.ProofOfWork.BlockHeight[i])
	}
	for i, v := range ppp.BlockHash {
		require.Equal(t, v, pp.ProofOfWork.BlockHash[i])
	}
	require.Equal(t, 3, len(ppp.SeenTx))
	require.Equal(t, p.SeenTx["1"], ppp.SeenTx["1"])
	require.Equal(t, p.SeenTx["2"], ppp.SeenTx["2"])
	require.Equal(t, p.SeenTx["3"], ppp.SeenTx["3"])
	require.Equal(t, p.SeenTid["100"], ppp.SeenTid["100"])
	require.Equal(t, p.SeenTid["200"], ppp.SeenTid["200"])
	require.Equal(t, p.SeenTid["300"], ppp.SeenTid["300"])

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
	require.Equal(t, uint64(105), ppp.BannedParties["party1"])
	require.Equal(t, uint64(104), ppp.BannedParties["party2"])
}

func TestSnapshot(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	e.OnEpochEvent(context.Background(), types.Epoch{Seq: 1, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.BeginBlock(100, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4")

	party := crypto.RandomHash()

	// difficulty 20 - 00000e31f8ac983354f5885d46b7631bc75f69ec82e8f6178bae53db0ab7e054
	require.NoError(t, e.DeliverTx(&testTx{txID: "1", party: party, blockHeight: 100, powTxID: "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F", powNonce: 424517}))

	// difficulty 21 but make it 20 - 000009c5043c4e1dd7fe190ece8d3fd83d94c4e2a2b7800456ce5f5a653c9f75
	require.NoError(t, e.DeliverTx(&testTx{txID: "2", party: party, blockHeight: 100, powTxID: "2A1319636230740888C968E4E7610D6DE820E644EEC3C08AA5322A0A022014BD", powNonce: 1421231}))

	// difficulty 22 - 000002a98320df372412d7179ca2645b13ff3ecbe660e4a9a743fb423d8aec1f
	require.NoError(t, e.DeliverTx(&testTx{txID: "3", party: party, blockHeight: 100, powTxID: "5B0E1EB96CCAC120E6D824A5F4C4007EABC59573B861BD84B1EF09DFB376DC84", powNonce: 4031737}))

	// difficulty 23 - 000001c297318619efd60b9197f89e36fea83ca8d7461cf7b7c78af84e0a3b51
	require.NoError(t, e.DeliverTx(&testTx{txID: "4", party: party, blockHeight: 100, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336}))

	e.BeginBlock(101, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F")
	e.BeginBlock(102, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F")

	key := (&types.PayloadProofOfWork{}).Key()
	hash1, err := e.GetHash(key)
	require.NoError(t, err)

	state1, _, err := e.GetState(key)
	require.NoError(t, err)

	eLoaded := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	eLoaded.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(1))
	eLoaded.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	eLoaded.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	eLoaded.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state1, &pl))
	eLoaded.LoadState(context.Background(), gtypes.PayloadFromProto(&pl))

	hash2, err := eLoaded.GetHash(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(hash1, hash2))

	state2, _, err := eLoaded.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state1, state2))
}
