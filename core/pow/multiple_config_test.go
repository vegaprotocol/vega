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

package pow

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/stretchr/testify/require"
)

func TestConfigurationHistory(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig())

	e.BeginBlock(1, crypto.RandomHash(), []abci.Tx{})
	require.Equal(t, 0, len(e.activeParams))
	require.Equal(t, 0, len(e.activeStates))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(1))
	require.Equal(t, 1, len(e.activeParams))
	require.Equal(t, 1, len(e.activeStates))
	require.Equal(t, uint64(1), e.activeParams[0].fromBlock)
	require.Nil(t, e.activeParams[0].untilBlock)

	e.BeginBlock(2, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(5))
	require.Equal(t, 2, len(e.activeParams))
	require.Equal(t, 2, len(e.activeStates))
	require.Equal(t, uint64(1), e.activeParams[0].fromBlock)
	require.Equal(t, uint64(2), *e.activeParams[0].untilBlock)
	require.Equal(t, uint64(3), e.activeParams[1].fromBlock)
	require.Nil(t, e.activeParams[1].untilBlock)

	// we're in block 2, we expect difficulty to change in block 3
	require.Equal(t, uint32(1), e.SpamPoWDifficulty())
	e.BeginBlock(3, crypto.RandomHash(), []abci.Tx{})
	require.Equal(t, uint32(5), e.SpamPoWDifficulty())
}

func TestSpamPoWNumberOfPastBlocksChange(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig())
	e.BeginBlock(1, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(10))

	// it's the first configuration so it's starting from this block rather than next
	require.Equal(t, uint32(10), e.SpamPoWNumberOfPastBlocks())
	e.BeginBlock(2, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(100))
	require.Equal(t, uint32(10), e.SpamPoWNumberOfPastBlocks())
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(110))
	require.Equal(t, uint32(10), e.SpamPoWNumberOfPastBlocks())
	e.BeginBlock(3, crypto.RandomHash(), []abci.Tx{})
	require.Equal(t, uint32(110), e.SpamPoWNumberOfPastBlocks())
}

func TestSpamPoWDifficultyChange(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig())
	e.BeginBlock(1, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(10))

	// it's the first configuration so it's starting from this block rather than next
	require.Equal(t, uint32(10), e.SpamPoWDifficulty())
	e.BeginBlock(2, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(100))
	require.Equal(t, uint32(10), e.SpamPoWDifficulty())
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(110))
	require.Equal(t, uint32(10), e.SpamPoWDifficulty())
	e.BeginBlock(3, crypto.RandomHash(), []abci.Tx{})
	require.Equal(t, uint32(110), e.SpamPoWDifficulty())
}

func TestSpamPoWHashChange(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig())
	e.BeginBlock(1, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWHashFunction(context.Background(), "f1")

	// it's the first configuration so it's starting from this block rather than next
	require.Equal(t, "f1", e.SpamPoWHashFunction())
	e.BeginBlock(2, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWHashFunction(context.Background(), "f2")
	require.Equal(t, "f1", e.SpamPoWHashFunction())
	e.UpdateSpamPoWHashFunction(context.Background(), "f3")
	require.Equal(t, "f1", e.SpamPoWHashFunction())
	e.BeginBlock(3, crypto.RandomHash(), []abci.Tx{})
	require.Equal(t, "f3", e.SpamPoWHashFunction())
}

func TestSpamPoWNumberOfTxPerBlockChange(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig())
	e.BeginBlock(1, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(10))

	// it's the first configuration so it's starting from this block rather than next
	require.Equal(t, uint32(10), e.SpamPoWNumberOfTxPerBlock())
	e.BeginBlock(2, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(100))
	require.Equal(t, uint32(10), e.SpamPoWNumberOfTxPerBlock())
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(110))
	require.Equal(t, uint32(10), e.SpamPoWNumberOfTxPerBlock())
	e.BeginBlock(3, crypto.RandomHash(), []abci.Tx{})
	require.Equal(t, uint32(110), e.SpamPoWNumberOfTxPerBlock())
}

func TestSpamPoWIncreasingDifficultyChange(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig())
	e.BeginBlock(1, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))

	// it's the first configuration so it's starting from this block rather than next
	require.Equal(t, true, e.SpamPoWIncreasingDifficulty())
	e.BeginBlock(2, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))
	require.Equal(t, true, e.SpamPoWIncreasingDifficulty())
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(0))
	require.Equal(t, true, e.SpamPoWIncreasingDifficulty())
	e.BeginBlock(3, crypto.RandomHash(), []abci.Tx{})
	require.Equal(t, false, e.SpamPoWIncreasingDifficulty())
}

func TestBlockData(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig())
	blockHash := crypto.RandomHash()
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(5))
	e.BeginBlock(1, blockHash, []abci.Tx{})

	height, hash := e.BlockData()
	require.Equal(t, uint64(1), height)
	require.Equal(t, blockHash, hash)

	blockHash = crypto.RandomHash()
	e.BeginBlock(2, blockHash, []abci.Tx{})
	height, hash = e.BlockData()
	require.Equal(t, uint64(2), height)
	require.Equal(t, blockHash, hash)
}

func TestFindParamsForBlockHeight(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig())
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(100))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), "sha3")
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))

	e.BeginBlock(9, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(21))
	e.BeginBlock(19, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(22))
	e.BeginBlock(29, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(23))

	require.Equal(t, uint(20), e.activeParams[e.findParamsForBlockHeight(1)].spamPoWDifficulty)
	require.Equal(t, uint(20), e.activeParams[e.findParamsForBlockHeight(9)].spamPoWDifficulty)
	require.Equal(t, uint(21), e.activeParams[e.findParamsForBlockHeight(10)].spamPoWDifficulty)
	require.Equal(t, uint(21), e.activeParams[e.findParamsForBlockHeight(19)].spamPoWDifficulty)
	require.Equal(t, uint(22), e.activeParams[e.findParamsForBlockHeight(20)].spamPoWDifficulty)
	require.Equal(t, uint(22), e.activeParams[e.findParamsForBlockHeight(29)].spamPoWDifficulty)
	require.Equal(t, uint(23), e.activeParams[e.findParamsForBlockHeight(30)].spamPoWDifficulty)
	require.Equal(t, uint(23), e.activeParams[e.findParamsForBlockHeight(100)].spamPoWDifficulty)
}

// func TestVerifyWithMultipleConfigs(t *testing.T) {
// 	e := New(logging.NewTestLogger(), NewDefaultConfig())
// 	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(100))
// 	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
// 	e.UpdateSpamPoWHashFunction(context.Background(), "sha3_24_rounds")
// 	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))
// 	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(0))

// 	block9Hash := "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4"
// 	e.BeginBlock(9, block9Hash)
// 	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(21))
// 	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(50))
// 	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(5))
// 	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))

// 	e.EndOfBlock()
// 	e.BeginBlock(19, crypto.RandomHash())
// 	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(22))
// 	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(80))
// 	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))
// 	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))

// 	e.EndOfBlock()
// 	block29Hash := "8890702af457ddcda01fba579a126adcecae954781500acb546fef9c8087a239"
// 	e.BeginBlock(29, block29Hash)
// 	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(1))
// 	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(70))
// 	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(2))
// 	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(0))
// 	e.EndOfBlock()

// 	block30Hash := "377EEAC9847D751A4FAFD3F2896E99C1A03363EBDA3036C33940CFE578E196D1"
// 	e.BeginBlock(30, block30Hash)

// 	// now we're in block 90
// 	e.BeginBlock(90, "792ca202b84226c739f9923046a0f4e7b5ff9e6f1b5636d8e26a8e2c5dec70ac")

// 	// transactions sent from block height < 10 have past blocks of 100, difficulty of 20 and allow 1 transaction per block with no increased difficulty
// 	tx9_1 := &testTx{txID: "1", blockHeight: 9, party: "party", powTxID: "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F", powNonce: 424517}
// 	err := e.DeliverTx(tx9_1)
// 	require.NoError(t, err)

// 	// doesn't really matter what the pow data is because in block 19 the parameters allowed only 50 blocks behind meaning in block 90 the transaction should be rejected as too old
// 	tx19 := &testTx{txID: "2", blockHeight: 19, party: "party", powTxID: "5B87F9DFA41DABE84A11CA78D9FE11DA8FC2AA926004CA66454A7AF0A206480D", powNonce: 4095356}
// 	err = e.DeliverTx(tx19)
// 	require.Equal(t, "unknown block height for tx:32, command:Amend Order, party:party", err.Error())

// 	// in block 29 we're allowed to submit 1 transactions with difficulty starting at 22 and increased difficulty
// 	tx29_1 := &testTx{txID: "3", blockHeight: 29, party: "party", powTxID: "74030ee7dc931be9d9cc5f2c9d44ac174b4144b377ef07a7bb1781856921dd43", powNonce: 1903233}
// 	tx29_2 := &testTx{txID: "4", blockHeight: 29, party: "party", powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 7914217}
// 	err = e.DeliverTx(tx29_1)
// 	require.NoError(t, err)
// 	err = e.DeliverTx(tx29_2)
// 	require.NoError(t, err)

// 	// finally we have a transaction sent in block 30 which allows for 2 transactions with no increased difficulty
// 	tx30_1 := &testTx{txID: "5", blockHeight: 30, party: "party", powTxID: "2A1319636230740888C968E4E7610D6DE820E644EEC3C08AA5322A0A022014BD", powNonce: 380742}
// 	err = e.DeliverTx(tx30_1)
// 	require.NoError(t, err)

// 	tx30_2 := &testTx{txID: "6", blockHeight: 30, party: "party", powTxID: "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", powNonce: 1296835}
// 	err = e.DeliverTx(tx30_2)
// 	require.NoError(t, err)
// 	tx30_3 := &testTx{txID: "7", blockHeight: 30, party: "party", powTxID: "5B0E1EB96CCAC120E6D824A5F4C4007EABC59573B861BD84B1EF09DFB376DC84", powNonce: 388948}
// 	err = e.DeliverTx(tx30_3)
// 	// the third transaction would be rejected and ban the party
// 	require.Equal(t, "too many transactions per block", err.Error())

// 	// now move on to the next block and the party should be banned
// 	e.BeginBlock(91, crypto.RandomHash())
// 	tx90 := &testTx{txID: "7", blockHeight: 90, party: "party", powTxID: "3b8399cdffee2686d75d1a96d22cd49cd11f62c93da20e72239895bfdaf4b772", powNonce: 0}
// 	err = e.DeliverTx(tx90)
// 	require.Equal(t, "party is banned from sending transactions", err.Error())
// }

func TestEndOfBlockCleanup(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig())
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(100))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), "sha3_24_rounds")
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(0))

	// the setting above is good for transactions from blocks 0 - 9+100
	// that means at the end of block 109 it can be removed

	e.BeginBlock(9, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(21))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(50))
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(5))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))

	// the setting above is good for transactions from blocks 10 - 19+50
	// that means at the end of block 69 it can be removed

	e.BeginBlock(19, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(22))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(80))
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))

	// the setting above is good for transactions from blocks 20 - 29+80
	// that means at the end of block 109 it can be removed

	e.BeginBlock(29, crypto.RandomHash(), []abci.Tx{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(70))
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(2))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(0))

	// the setting above is good for transactions from blocks 30 -

	e.BeginBlock(30, crypto.RandomHash(), []abci.Tx{})

	require.Equal(t, 4, len(e.activeParams))
	require.Equal(t, 4, len(e.activeStates))

	e.BeginBlock(69, crypto.RandomHash(), []abci.Tx{})

	require.Equal(t, 3, len(e.activeParams))
	require.Equal(t, 3, len(e.activeStates))

	e.BeginBlock(109, crypto.RandomHash(), []abci.Tx{})

	require.Equal(t, 1, len(e.activeParams))
	require.Equal(t, 1, len(e.activeStates))
	require.Equal(t, uint32(70), e.SpamPoWNumberOfPastBlocks())
	require.Equal(t, uint32(1), e.SpamPoWDifficulty())
	require.Equal(t, false, e.SpamPoWIncreasingDifficulty())
	require.Equal(t, uint32(2), e.SpamPoWNumberOfTxPerBlock())
}
