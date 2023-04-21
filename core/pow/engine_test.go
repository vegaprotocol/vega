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
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/core/delegation/mocks"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestUpdateBanDuration(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.OnEpochDurationChanged(context.Background(), time.Hour*10)
	require.Equal(t, float64(750), e.banDuration.Round(time.Second).Seconds()) // 10h/48 = 10 * 60 * 60 / 48 = 750s

	e.OnEpochDurationChanged(context.Background(), time.Second*10)
	require.Equal(t, float64(30), e.banDuration.Round(time.Second).Seconds()) // minimum of 30s applies
}

func TestSpamPoWNumberOfPastBlocks(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(200))
	require.Equal(t, uint32(200), e.SpamPoWNumberOfPastBlocks())
}

func TestSpamPoWDifficulty(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	require.Equal(t, uint32(20), e.SpamPoWDifficulty())
}

func TestSpamPoWHashFunction(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWHashFunction(context.Background(), "hash4")
	require.Equal(t, "hash4", e.SpamPoWHashFunction())
}

func TestSpamPoWNumberOfTxPerBlock(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(2))
	require.Equal(t, uint32(2), e.SpamPoWNumberOfPastBlocks())
}

func TestSpamPoWIncreasingDifficulty(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))
	require.Equal(t, true, e.SpamPoWIncreasingDifficulty())
}

func TestUpdateNumberOfBlocks(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(5))
	require.Equal(t, uint32(5), e.SpamPoWNumberOfPastBlocks())
}

func TestCheckTx(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(5))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	e.currentBlock = 100
	e.blockHeight[100] = 100
	e.blockHash[100] = "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9"
	e.seenTid["49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"] = struct{}{}
	e.heightToTid[96] = []string{"49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"}

	// seen transction
	require.Equal(t, errors.New("proof of work tid already used"), e.CheckTx(&testTx{blockHeight: 100, powTxID: "49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"}))

	// party is banned
	e.bannedParties["C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8"] = time.Time{}
	require.Equal(t, errors.New("party is banned from sending transactions"), e.CheckTx(&testTx{party: "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8", blockHeight: 100, powTxID: "A204DF39B63100C76EC831A843BF3C538FF54217DBA4B1409A3773507053EBB5"}))

	// incorrect pow
	require.Equal(t, errors.New("failed to verify proof of work"), e.CheckTx(&testTx{party: crypto.RandomHash(), blockHeight: 100, powTxID: "077723AB0705677EAA704130D403C21352F87A9AF0E9C4C8F85CC13245FEFED7", powNonce: 1}))

	// all good
	require.NoError(t, e.CheckTx(&testTx{party: crypto.RandomHash(), blockHeight: 100, powTxID: "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", powNonce: 596}))
}

func TestCheckTxValidator(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(5))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	e.currentBlock = 100
	e.blockHeight[100] = 100
	e.blockHash[100] = "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9"
	e.seenTid["49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"] = struct{}{}
	e.heightToTid[96] = []string{"49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"}

	// seen transction
	require.Equal(t, errors.New("proof of work tid already used"), e.CheckTx(&testTx{blockHeight: 100, powTxID: "49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"}))

	// party is banned
	e.bannedParties["C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8"] = time.Time{}

	// transaction too old: height 10, number of past blocks 5, current block 100
	oldTx := &testTx{
		party:       crypto.RandomHash(),
		blockHeight: 10,
		powTxID:     "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4",
		powNonce:    596,
	}
	txHash := hex.EncodeToString(oldTx.Hash())
	expErr := errors.New("unknown block height for tx:" + txHash + ", command:" + oldTx.Command().String() + ", party:" + oldTx.Party())
	require.Equal(t, expErr, e.CheckTx(oldTx))
	// old tx, but validator command still is good.
	require.NoError(t, e.CheckTx(&testValidatorTx{testTx: *oldTx}))
}

func TestDeliverTx(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(5))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	e.currentBlock = 100
	e.blockHeight[100] = 100
	e.blockHash[100] = "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9"

	require.Equal(t, 0, len(e.seenTid))
	require.Equal(t, 0, len(e.heightToTid))
	party := crypto.RandomHash()
	require.NoError(t, e.DeliverTx(&testTx{party: party, blockHeight: 100, powTxID: "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", powNonce: 596}))
	require.Equal(t, 1, len(e.seenTid))
	require.Equal(t, 1, len(e.heightToTid))
	require.Equal(t, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", e.heightToTid[100][0])
}

func TestMempoolTidRejection(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(5))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	party := crypto.RandomHash()
	e.currentBlock = 100
	e.blockHeight[100] = 100
	e.blockHash[100] = "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9"

	tx1 := &testTx{party: party, blockHeight: 100, powTxID: "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", powNonce: 596}
	require.NoError(t, e.CheckTx(tx1))
	require.Equal(t, 1, len(e.mempoolSeenTid))

	require.Error(t, e.CheckTx(tx1))
	e.DeliverTx(tx1)

	require.Equal(t, 1, len(e.seenTid))
	_, ok := e.seenTid["2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4"]
	require.True(t, ok)

	e.Commit()
	require.Equal(t, 0, len(e.mempoolSeenTid))

	require.Error(t, e.CheckTx(tx1))
}

func TestExpectedDifficulty(t *testing.T) {
	type args struct {
		spamPowDifficulty         uint
		spamPoWNumberOfTxPerBlock uint
		seenTx                    uint
	}

	tests := []struct {
		name           string
		args           args
		wantTotal      uint
		wantDifficulty uint
	}{
		{
			name: "3 transactions",
			args: args{
				spamPowDifficulty:         20,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    3,
			},
			wantTotal:      60, // 3 * 20
			wantDifficulty: 20,
		},
		{
			name: "5 transactions",
			args: args{
				spamPowDifficulty:         20,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    5,
			},
			wantTotal:      100, // 5 * 20
			wantDifficulty: 21,
		},
		{
			name: "6 transactions",
			args: args{
				spamPowDifficulty:         20,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    6,
			},
			wantTotal:      121, // 5 * 20 + 21
			wantDifficulty: 21,
		},
		{
			name: "9 transactions",
			args: args{
				spamPowDifficulty:         20,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    9,
			},
			wantTotal:      184, // 5 * 20 + 4 * 21
			wantDifficulty: 21,
		},
		{
			name: "10 transactions",
			args: args{
				spamPowDifficulty:         20,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    10,
			},
			wantTotal:      205, // 5 * 20 + 5 * 21
			wantDifficulty: 22,
		},
		{
			name: "20 transactions",
			args: args{
				spamPowDifficulty:         20,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    20,
			},
			wantTotal:      430, // 5 * 20 + 5 * 21 + 5 * 22 + 5 * 23
			wantDifficulty: 24,
		},
		{
			name: "22 transactions",
			args: args{
				spamPowDifficulty:         20,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    22,
			},
			wantTotal:      478, // 5 * 20 + 5 * 21 + 5 * 22 + 5 * 23 + 2 * 24
			wantDifficulty: 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTotal, gotDifficulty := calculateExpectedDifficulty(tt.args.spamPowDifficulty, tt.args.spamPoWNumberOfTxPerBlock, tt.args.seenTx)
			require.Equal(t, tt.wantTotal, gotTotal)
			require.Equal(t, tt.wantDifficulty, gotDifficulty)
		})
	}
}

func TestBeginBlock(t *testing.T) {
	ts := mocks.NewMockTimeService(gomock.NewController(t))
	ts.EXPECT().GetTimeNow().AnyTimes().Return(time.Now())
	e := New(logging.NewTestLogger(), NewDefaultConfig(), ts)
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(3))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	e.BeginBlock(100, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4")
	e.BeginBlock(101, "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9")
	e.BeginBlock(102, "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8")

	require.Equal(t, uint64(102), e.currentBlock)
	require.Equal(t, uint64(100), e.blockHeight[100])
	require.Equal(t, uint64(101), e.blockHeight[101])
	require.Equal(t, uint64(102), e.blockHeight[102])
	require.Equal(t, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", e.blockHash[100])
	require.Equal(t, "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9", e.blockHash[101])
	require.Equal(t, "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8", e.blockHash[102])

	// now add some transactions for block 100 before it goes off
	e.DeliverTx(&testTx{txID: "1", party: crypto.RandomHash(), blockHeight: 100, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336})
	e.DeliverTx(&testTx{txID: "2", party: crypto.RandomHash(), blockHeight: 100, powTxID: "DC911C0EA95545441F3E1182DD25D973764395A7E75CBDBC086F1C6F7075AED6", powNonce: 523162})

	require.Equal(t, 2, len(e.seenTid))
	require.Equal(t, 2, len(e.heightToTid[100]))

	e.BeginBlock(103, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F")
	require.Equal(t, uint64(103), e.currentBlock)
	require.Equal(t, uint64(103), e.blockHeight[103])
	require.Equal(t, uint64(101), e.blockHeight[101])
	require.Equal(t, uint64(102), e.blockHeight[102])
	require.Equal(t, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F", e.blockHash[103])
	require.Equal(t, "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9", e.blockHash[101])
	require.Equal(t, "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8", e.blockHash[102])
}

func TestBan(t *testing.T) {
	now := time.Now()
	ts := mocks.NewMockTimeService(gomock.NewController(t))
	ts.EXPECT().GetTimeNow().Times(72).Return(now)
	e := New(logging.NewTestLogger(), NewDefaultConfig(), ts)
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))
	e.OnEpochDurationChanged(context.Background(), 24*time.Hour)

	// test happy days first - 4 transactions with increasing difficulty results in no ban - regardless of the order they come in
	party := crypto.RandomHash()
	txs := []*testTx{
		{txID: "4", party: party, powTxID: "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F", powNonce: 424517},  // 00000e31f8ac983354f5885d46b7631bc75f69ec82e8f6178bae53db0ab7e054 - 20
		{txID: "5", party: party, powTxID: "5B87F9DFA41DABE84A11CA78D9FE11DA8FC2AA926004CA66454A7AF0A206480D", powNonce: 4095356}, // 0000077b7d66117b57e45ccba0c31554e61c9853cc1cd9a2cf09c41b0aa9c22e - 21
		{txID: "6", party: party, powTxID: "B14DD602ED48C9F7B5367105A4A97FFC9199EA0C9E1490B786534768DD1538EF", powNonce: 1751582}, // 000003bbf0cde49e3899ad23282b18defbc12a65f07c95d768464b87024df368 - 22
		{txID: "7", party: party, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336},  // 000001c297318619efd60b9197f89e36fea83ca8d7461cf7b7c78af84e0a3b51 - 23
	}
	testBanWithTxPermutations(t, e, txs, false, 102, party, now)

	txs = []*testTx{
		{txID: "8", party: party, powTxID: "2A1319636230740888C968E4E7610D6DE820E644EEC3C08AA5322A0A022014BD", powNonce: 1421231},  // 000009c5043c4e1dd7fe190ece8d3fd83d94c4e2a2b7800456ce5f5a653c9f75 - 20
		{txID: "9", party: party, powTxID: "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F", powNonce: 424517},   // 00000e31f8ac983354f5885d46b7631bc75f69ec82e8f6178bae53db0ab7e054 - 20
		{txID: "10", party: party, powTxID: "5B0E1EB96CCAC120E6D824A5F4C4007EABC59573B861BD84B1EF09DFB376DC84", powNonce: 4031737}, // 000002a98320df372412d7179ca2645b13ff3ecbe660e4a9a743fb423d8aec1f - 22
		{txID: "11", party: party, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336},  // 000001c297318619efd60b9197f89e36fea83ca8d7461cf7b7c78af84e0a3b51 - 23
	}
	testBanWithTxPermutations(t, e, txs, true, 126, party, now)
	now = now.Add(e.banDuration)
	ts.EXPECT().GetTimeNow().Times(1).Return(now)
	e.BeginBlock(129, crypto.RandomHash())
	require.Equal(t, 0, len(e.bannedParties))
}

func TestAllowTransactionsAcrossMultipleBlocks(t *testing.T) {
	now := time.Now()
	ts := mocks.NewMockTimeService(gomock.NewController(t))
	ts.EXPECT().GetTimeNow().AnyTimes().Return(now)
	e := New(logging.NewTestLogger(), NewDefaultConfig(), ts)
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(10))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))
	e.BeginBlock(100, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4")

	// test happy days first - 4 transactions with increasing difficulty results in no ban - regardless of the order they come in
	party := crypto.RandomHash()
	txs := []*testTx{
		{txID: "9", blockHeight: 100, party: party, powTxID: "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F", powNonce: 424517},   // 00000e31f8ac983354f5885d46b7631bc75f69ec82e8f6178bae53db0ab7e054 - 20
		{txID: "10", blockHeight: 100, party: party, powTxID: "5B0E1EB96CCAC120E6D824A5F4C4007EABC59573B861BD84B1EF09DFB376DC84", powNonce: 4031737}, // 000002a98320df372412d7179ca2645b13ff3ecbe660e4a9a743fb423d8aec1f - 22
		{txID: "11", blockHeight: 100, party: party, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336},  // 000001c297318619efd60b9197f89e36fea83ca8d7461cf7b7c78af84e0a3b51 - 23
		{txID: "8", blockHeight: 100, party: party, powTxID: "2A1319636230740888C968E4E7610D6DE820E644EEC3C08AA5322A0A022014BD", powNonce: 1421231},  // 000009c5043c4e1dd7fe190ece8d3fd83d94c4e2a2b7800456ce5f5a653c9f75 - 20
	}

	// process the first transaction on block 101
	e.BeginBlock(101, crypto.RandomHash())
	require.NoError(t, e.DeliverTx(txs[0]))

	// process the second transaction on block 102
	e.BeginBlock(102, crypto.RandomHash())
	require.NoError(t, e.DeliverTx(txs[1]))

	// process the third transaction on block 103
	e.BeginBlock(103, crypto.RandomHash())
	require.NoError(t, e.DeliverTx(txs[2]))

	// process the last transaction on block 104 - this should get us banned
	e.BeginBlock(104, crypto.RandomHash())
	require.Equal(t, "too many transactions per block", e.DeliverTx(txs[3]).Error())

	require.Equal(t, 1, len(e.bannedParties))
}

func TestEndBlock(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), mocks.NewMockTimeService(gomock.NewController(t)))
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))
}

func testBanWithTxPermutations(t *testing.T, e *Engine, txs []*testTx, expectedBan bool, blockHeight uint64, party string, now time.Time) {
	t.Helper()
	txsPerm := permutation(txs)
	for i, perm := range txsPerm {
		// clear any bans
		e.bannedParties = map[string]time.Time{}

		// begin a new block
		e.BeginBlock(blockHeight+uint64(i), "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4")

		// send the transactions with the given permutation
		errors := []error{}
		for _, p := range perm {
			p.blockHeight = blockHeight + uint64(i)
			err := e.DeliverTx(p)
			if err != nil {
				errors = append(errors, err)
			}
		}

		if expectedBan {
			require.True(t, len(errors) > 0)
			require.Equal(t, "too many transactions per block", errors[0].Error())
		} else {
			require.Equal(t, 0, len(errors))
		}

		// end the block to check if the party was playing nice
		e.EndOfBlock()

		// verify expected ban
		if expectedBan {
			require.Equal(t, 1, len(e.bannedParties))
			require.Equal(t, now.Add(e.banDuration), e.bannedParties[party])
		} else {
			require.Equal(t, 0, len(e.bannedParties))
		}
	}
}

func permutation(xs []*testTx) (permuts [][]*testTx) {
	var rc func([]*testTx, int)
	rc = func(a []*testTx, k int) {
		if k == len(a) {
			permuts = append(permuts, append([]*testTx{}, a...))
		} else {
			for i := k; i < len(xs); i++ {
				a[k], a[i] = a[i], a[k]
				rc(a, k+1)
				a[k], a[i] = a[i], a[k]
			}
		}
	}
	rc(xs, 0)

	return permuts
}

type testTx struct {
	party       string
	blockHeight uint64
	powNonce    uint64
	powTxID     string
	txID        string
}

type testValidatorTx struct {
	testTx
}

func (tx *testTx) Unmarshal(interface{}) error { return nil }
func (tx *testTx) GetPoWTID() string           { return tx.powTxID }
func (tx *testTx) GetVersion() uint32          { return 2 }
func (tx *testTx) GetPoWNonce() uint64         { return tx.powNonce }
func (tx *testTx) Signature() []byte           { return []byte{} }
func (tx *testTx) Payload() []byte             { return []byte{} }
func (tx *testTx) PubKey() []byte              { return []byte{} }
func (tx *testTx) PubKeyHex() string           { return "" }
func (tx *testTx) Party() string               { return tx.party }
func (tx *testTx) Hash() []byte                { return []byte(tx.txID) }
func (tx *testTx) Command() txn.Command        { return txn.AmendOrderCommand }
func (tx *testTx) BlockHeight() uint64         { return tx.blockHeight }
func (tx *testTx) GetCmd() interface{}         { return nil }
func (tx *testTx) Validate() error             { return nil }

func (tx *testValidatorTx) Command() txn.Command {
	return txn.NodeSignatureCommand
}

func Test_ExpectedSpamDifficulty(t *testing.T) {
	type args struct {
		spamPowDifficulty         uint
		spamPoWNumberOfTxPerBlock uint
		seenTx                    uint
		observedDifficulty        uint
		increaseDifficulty        bool
	}

	tests := []struct {
		name  string
		args  args
		isNil bool
		want  uint64
	}{
		{
			name: "Expected difficulty after 12 txs",
			args: args{
				spamPowDifficulty:         10,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    12,
				observedDifficulty:        132,
				increaseDifficulty:        true,
			},
			isNil: false,
			want:  10,
		},
		{
			name: "Expected difficulty after 13 txs",
			args: args{
				spamPowDifficulty:         10,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    13,
				observedDifficulty:        142,
				increaseDifficulty:        true,
			},
			isNil: false,
			want:  11,
		},
		{
			name: "Expected difficulty after 14 txs",
			args: args{
				spamPowDifficulty:         10,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    14,
				observedDifficulty:        153,
				increaseDifficulty:        true,
			},
			isNil: false,
			want:  12,
		},
		{
			name: "Expected difficulty after 15 txs",
			args: args{
				spamPowDifficulty:         10,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    15,
				observedDifficulty:        166,
				increaseDifficulty:        true,
			},
			isNil: false,
			want:  12, // after 15txs, the difficulty is increased to 13, but we should have 1 extra in the credit from the previous block
		},
		{
			name: "Expected difficulty after 16 txs",
			args: args{
				spamPowDifficulty:         10,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    16,
				observedDifficulty:        178,
				increaseDifficulty:        true,
			},
			isNil: false,
			want:  13,
		},
		{
			name: "Expected difficulty after 17 txs",
			args: args{
				spamPowDifficulty:         10,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    17,
				observedDifficulty:        193,
				increaseDifficulty:        true,
			},
			isNil: false,
			want:  11,
		},
		{
			name: "Expected difficulty when increaseDifficulty is false",
			args: args{
				spamPowDifficulty:         10,
				spamPoWNumberOfTxPerBlock: 5,
				seenTx:                    17,
				observedDifficulty:        193,
				increaseDifficulty:        false,
			},
			isNil: true,
		},
		{
			name: "Expected difficulty when increaseDifficulty is false but fewer seen than allowed in block",
			args: args{
				spamPowDifficulty:         10,
				spamPoWNumberOfTxPerBlock: 100,
				seenTx:                    1,
				observedDifficulty:        10,
				increaseDifficulty:        false,
			},
			isNil: false,
			want:  10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMinDifficultyForNextTx(
				tt.args.spamPowDifficulty,
				tt.args.spamPoWNumberOfTxPerBlock,
				tt.args.seenTx,
				tt.args.observedDifficulty,
				tt.args.increaseDifficulty)
			if tt.isNil {
				assert.Nil(t, got)
				return
			}

			assert.Equal(t, tt.want, *got, "getMinDifficultyForNextTx() = %v, want %v", *got, tt.want)
		})
	}
}
