package pow

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/shared/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
)

type TestEpochEngine struct {
	callbacks []func(context.Context, types.Epoch)
	restore   []func(context.Context, types.Epoch)
}

func (e *TestEpochEngine) NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch)) {
	e.callbacks = append(e.callbacks, f)
	e.restore = append(e.restore, r)
}

func TestSpamPoWNumberOfPastBlocks(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(200))
	require.Equal(t, uint32(200), e.SpamPoWNumberOfPastBlocks())
}

func TestSpamPoWDifficulty(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	require.Equal(t, uint32(20), e.SpamPoWDifficulty())
}

func TestSpamPoWHashFunction(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWHashFunction(context.Background(), "hash4")
	require.Equal(t, "hash4", e.SpamPoWHashFunction())
}

func TestSpamPoWNumberOfTxPerBlock(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(2))
	require.Equal(t, uint32(2), e.SpamPoWNumberOfPastBlocks())
}

func TestSpamPoWIncreasingDifficulty(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))
	require.Equal(t, true, e.SpamPoWIncreasingDifficulty())

	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.Zero())
	require.Equal(t, false, e.SpamPoWIncreasingDifficulty())
}

func TestIncreaseNumberOfBlocks(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(5))
	e.currentBlock = 100
	e.heightToTid[100] = []string{
		"C4F57F3B29558F4FF004EA8F68D218A2464F6BAC3800E7DAF681E394D65BB915",
	}
	e.heightToTid[99] = []string{
		"8B7488CC0CED49E33D0281C871A3242C4D6A64EDF0392B629A0D3CBEE6ECEBE2",
		"F3BCD061A8B623569FA372D81BB4FA41CCCEA939EF40C47B6A72625A13768EA7",
	}
	e.heightToTid[98] = []string{
		"916F09307A7DDAF53F87CC102235A77E0BA5503BAAFF5C14A35A8C93D49883C5",
		"B6413228EEA5B3AD2FB93F8E1DBCFDE9417913A4AC22466C70DA2AEBAD3A66B6",
		"C3B5372C2FDAE61C45A8C7C87452E9542338A79A08B38F8BD66383D11551FCBD",
	}
	e.heightToTid[97] = []string{
		"E22CBC3D5CD9B38636621EAE3D3AFC3CF6F1021570E9B02CB388250DA0B473D8",
		"06188E5E7B5E7DAD02C4FC513E13D5E87DFE543C67754531094940F46DD8CEF6",
		"7ED74C389C4C4C7E53F4AF7559FEE627FCD070A1B9273B35116F4BC43265B8F5",
		"C429C145E87BECEE6203D088E80FEDD10BB7716C894CEE191F650FAE517B5F0F",
	}
	e.heightToTid[96] = []string{
		"158180D4206CD6EE875BBD376EF2509DF26C164232D5237D04398F96A5A6126A",
		"B13B2B006DEAC952488EDCEB2A8D4FFB29A2A1D7FC251D22FB519B8EC530257B",
		"93306D50C49881D456749DB48D4805CC161CDD567A361E5A7C9F94C1FCC5F3D3",
		"F9D52D54C0D82939A2E95DA26AEA5AE5B677D63F98EC004A67B011FC08253012",
		"2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F",
	}

	for _, v := range e.heightToTid {
		for _, tx := range v {
			e.seenTid[tx] = struct{}{}
		}
	}

	e.blockHeight[96%5] = 96
	e.blockHeight[97%5] = 97
	e.blockHeight[98%5] = 98
	e.blockHeight[99%5] = 99
	e.blockHeight[100%5] = 100

	e.blockHash[96%5] = "A204DF39B63100C76EC831A843BF3C538FF54217DBA4B1409A3773507053EBB5"
	e.blockHash[97%5] = "077723AB0705677EAA704130D403C21352F87A9AF0E9C4C8F85CC13245FEFED7"
	e.blockHash[98%5] = "49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"
	e.blockHash[99%5] = "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9"
	e.blockHash[100%5] = "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8"

	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(10))
	require.Equal(t, uint32(10), e.SpamPoWNumberOfPastBlocks())
	require.Equal(t, 10, len(e.blockHash))
	require.Equal(t, 10, len(e.blockHeight))
	require.Equal(t, 5, len(e.heightToTid))
	require.Equal(t, 15, len(e.seenTid))

	require.Equal(t, uint64(100), e.blockHeight[0])
	require.Equal(t, uint64(0), e.blockHeight[1])
	require.Equal(t, uint64(0), e.blockHeight[2])
	require.Equal(t, uint64(0), e.blockHeight[3])
	require.Equal(t, uint64(0), e.blockHeight[4])
	require.Equal(t, uint64(0), e.blockHeight[5])
	require.Equal(t, uint64(96), e.blockHeight[6])
	require.Equal(t, uint64(97), e.blockHeight[7])
	require.Equal(t, uint64(98), e.blockHeight[8])
	require.Equal(t, uint64(99), e.blockHeight[9])

	require.Equal(t, "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8", e.blockHash[0])
	require.Equal(t, "", e.blockHash[1])
	require.Equal(t, "", e.blockHash[2])
	require.Equal(t, "", e.blockHash[3])
	require.Equal(t, "", e.blockHash[4])
	require.Equal(t, "", e.blockHash[5])
	require.Equal(t, "A204DF39B63100C76EC831A843BF3C538FF54217DBA4B1409A3773507053EBB5", e.blockHash[6])
	require.Equal(t, "077723AB0705677EAA704130D403C21352F87A9AF0E9C4C8F85CC13245FEFED7", e.blockHash[7])
	require.Equal(t, "49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB", e.blockHash[8])
	require.Equal(t, "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9", e.blockHash[9])

	require.Equal(t, 1, len(e.heightToTid[100]))
	require.Equal(t, 2, len(e.heightToTid[99]))
	require.Equal(t, 3, len(e.heightToTid[98]))
	require.Equal(t, 4, len(e.heightToTid[97]))
	require.Equal(t, 5, len(e.heightToTid[96]))
}

func TestDecreaseNumberOfBlocks(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(5))
	e.currentBlock = 100
	e.heightToTid[100] = []string{
		"C4F57F3B29558F4FF004EA8F68D218A2464F6BAC3800E7DAF681E394D65BB915",
	}
	e.heightToTid[99] = []string{
		"8B7488CC0CED49E33D0281C871A3242C4D6A64EDF0392B629A0D3CBEE6ECEBE2",
		"F3BCD061A8B623569FA372D81BB4FA41CCCEA939EF40C47B6A72625A13768EA7",
	}
	e.heightToTid[98] = []string{
		"916F09307A7DDAF53F87CC102235A77E0BA5503BAAFF5C14A35A8C93D49883C5",
		"B6413228EEA5B3AD2FB93F8E1DBCFDE9417913A4AC22466C70DA2AEBAD3A66B6",
		"C3B5372C2FDAE61C45A8C7C87452E9542338A79A08B38F8BD66383D11551FCBD",
	}
	e.heightToTid[97] = []string{
		"E22CBC3D5CD9B38636621EAE3D3AFC3CF6F1021570E9B02CB388250DA0B473D8",
		"06188E5E7B5E7DAD02C4FC513E13D5E87DFE543C67754531094940F46DD8CEF6",
		"7ED74C389C4C4C7E53F4AF7559FEE627FCD070A1B9273B35116F4BC43265B8F5",
		"C429C145E87BECEE6203D088E80FEDD10BB7716C894CEE191F650FAE517B5F0F",
	}
	e.heightToTid[96] = []string{
		"158180D4206CD6EE875BBD376EF2509DF26C164232D5237D04398F96A5A6126A",
		"B13B2B006DEAC952488EDCEB2A8D4FFB29A2A1D7FC251D22FB519B8EC530257B",
		"93306D50C49881D456749DB48D4805CC161CDD567A361E5A7C9F94C1FCC5F3D3",
		"F9D52D54C0D82939A2E95DA26AEA5AE5B677D63F98EC004A67B011FC08253012",
		"2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F",
	}

	for _, v := range e.heightToTid {
		for _, tx := range v {
			e.seenTid[tx] = struct{}{}
		}
	}

	e.blockHeight[96%5] = 96
	e.blockHeight[97%5] = 97
	e.blockHeight[98%5] = 98
	e.blockHeight[99%5] = 99
	e.blockHeight[100%5] = 100

	e.blockHash[96%5] = "A204DF39B63100C76EC831A843BF3C538FF54217DBA4B1409A3773507053EBB5"
	e.blockHash[97%5] = "077723AB0705677EAA704130D403C21352F87A9AF0E9C4C8F85CC13245FEFED7"
	e.blockHash[98%5] = "49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"
	e.blockHash[99%5] = "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9"
	e.blockHash[100%5] = "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8"

	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(3))

	require.Equal(t, uint32(3), e.SpamPoWNumberOfPastBlocks())
	require.Equal(t, 3, len(e.blockHash))
	require.Equal(t, 3, len(e.blockHeight))
	require.Equal(t, 3, len(e.heightToTid))
	require.Equal(t, 6, len(e.seenTid))

	require.Equal(t, uint64(99), e.blockHeight[0])
	require.Equal(t, uint64(100), e.blockHeight[1])
	require.Equal(t, uint64(98), e.blockHeight[2])

	require.Equal(t, "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9", e.blockHash[0])
	require.Equal(t, "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8", e.blockHash[1])
	require.Equal(t, "49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB", e.blockHash[2])
	require.Equal(t, 1, len(e.heightToTid[100]))
	require.Equal(t, 2, len(e.heightToTid[99]))
	require.Equal(t, 3, len(e.heightToTid[98]))
	require.Equal(t, 0, len(e.heightToTid[97]))
	require.Equal(t, 0, len(e.heightToTid[96]))
}

func TestCheckTx(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(5))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	// incorrect block
	require.Equal(t, errors.New("unknown block height"), e.CheckTx(&testTx{blockHeight: 100}))

	e.currentBlock = 100
	e.blockHeight[0] = 100
	e.blockHash[0] = "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9"
	e.seenTid["49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"] = struct{}{}
	e.heightToTid[96] = []string{"49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"}

	// seen transction
	require.Equal(t, errors.New("transaction ID already used"), e.CheckTx(&testTx{blockHeight: 100, powTxID: "49B0DF0954A8C048554B1C65F4F5883C38640D101A11959EB651AE2065A80BBB"}))

	// party is banned
	e.bannedParties["C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8"] = 6
	require.Equal(t, errors.New("party is banned from sending transactions"), e.CheckTx(&testTx{party: "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8", blockHeight: 100, powTxID: "A204DF39B63100C76EC831A843BF3C538FF54217DBA4B1409A3773507053EBB5"}))

	// incorrect pow
	require.Equal(t, errors.New("failed to verify proof of work"), e.CheckTx(&testTx{party: crypto.RandomHash(), blockHeight: 100, powTxID: "077723AB0705677EAA704130D403C21352F87A9AF0E9C4C8F85CC13245FEFED7", powNonce: 1}))

	// all good
	require.NoError(t, e.CheckTx(&testTx{party: crypto.RandomHash(), blockHeight: 100, powTxID: "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", powNonce: 596}))
}

func TestDeliverTx(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(5))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	e.currentBlock = 100
	e.blockHeight[0] = 100
	e.blockHash[0] = "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9"

	require.Equal(t, 0, len(e.seenTid))
	require.Equal(t, 0, len(e.heightToTid))
	party := crypto.RandomHash()
	require.NoError(t, e.DeliverTx(&testTx{party: party, blockHeight: 100, powTxID: "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", powNonce: 596}))
	require.Equal(t, 1, len(e.seenTid))
	require.Equal(t, 1, len(e.heightToTid))
	require.Equal(t, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", e.heightToTid[100][0])
	require.Equal(t, "54152512217159022870910287014126436409543381468363031094569505280365618", e.blockPartyToPoW[party][0].String())
}

func TestBeginBlock(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(3))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	e.BeginBlock(100, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4")
	e.BeginBlock(101, "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9")
	e.BeginBlock(102, "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8")

	require.Equal(t, uint64(102), e.currentBlock)
	require.Equal(t, uint64(100), e.blockHeight[1])
	require.Equal(t, uint64(101), e.blockHeight[2])
	require.Equal(t, uint64(102), e.blockHeight[0])
	require.Equal(t, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", e.blockHash[1])
	require.Equal(t, "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9", e.blockHash[2])
	require.Equal(t, "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8", e.blockHash[0])

	// now add some transactions for block 100 before it goes off
	e.DeliverTx(&testTx{txID: "1", party: crypto.RandomHash(), blockHeight: 100, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336})
	e.DeliverTx(&testTx{txID: "2", party: crypto.RandomHash(), blockHeight: 100, powTxID: "DC911C0EA95545441F3E1182DD25D973764395A7E75CBDBC086F1C6F7075AED6", powNonce: 523162})

	require.Equal(t, 2, len(e.seenTid))
	require.Equal(t, 2, len(e.heightToTid[100]))

	e.BeginBlock(103, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F")
	require.Equal(t, uint64(103), e.currentBlock)
	require.Equal(t, uint64(103), e.blockHeight[1])
	require.Equal(t, uint64(101), e.blockHeight[2])
	require.Equal(t, uint64(102), e.blockHeight[0])
	require.Equal(t, "2E289FB9CEF7234E2C08F34CCD66B330229067CE47E22F76EF0595B3ABA9968F", e.blockHash[1])
	require.Equal(t, "113EB390CBEB921433BDBA832CCDFD81AC4C77C3748A41B1AF08C96BC6C7BCD9", e.blockHash[2])
	require.Equal(t, "C692100485479CE9E1815B9E0A66D3596295A04DB42170CB4B61CFAE7332ADD8", e.blockHash[0])

	require.Equal(t, 0, len(e.seenTid))
	require.Equal(t, 0, len(e.heightToTid))
}

func TestEndBlock(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	party := crypto.RandomHash()
	e.BeginBlock(100, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4")
	e.DeliverTx(&testTx{txID: "1", party: party, blockHeight: 100, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336})
	require.Equal(t, 1, len(e.blockPartyToPoW))
	require.Equal(t, 0, len(e.bannedParties))

	// we only sent one transaction for the block and it's legit with the difficulty so we expect no banning in the end of block
	e.EndOfBlock()
	require.Equal(t, 0, len(e.bannedParties))
	require.Equal(t, 0, len(e.blockPartyToPoW))

	// now lets submit 2 transactions both with the same difficulty for the same block (will reuse the same block hash for simplicity)
	// this should cause the party to be banned because increase difficulty is off.
	e.BeginBlock(101, "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4")
	require.NoError(t, e.DeliverTx(&testTx{txID: "2", party: party, blockHeight: 101, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336}))
	require.NoError(t, e.DeliverTx(&testTx{txID: "3", party: party, blockHeight: 101, powTxID: "DC911C0EA95545441F3E1182DD25D973764395A7E75CBDBC086F1C6F7075AED6", powNonce: 523162}))

	require.Equal(t, 2, len(e.blockPartyToPoW[party]))
	e.EndOfBlock()
	require.Equal(t, 1, len(e.bannedParties))
	require.Equal(t, uint64(4), e.bannedParties[party])
	require.Equal(t, 0, len(e.blockPartyToPoW))

	// enable increasing difficulty
	e.UpdateSpamPoWIncreasingDifficulty(context.Background(), num.NewUint(1))

	// test happy days first - 4 transactions with increasing difficulty results in no ban - regardless of the order they come in
	txs := []*testTx{
		{txID: "4", party: party, powTxID: "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F", powNonce: 424517},  // 00000e31f8ac983354f5885d46b7631bc75f69ec82e8f6178bae53db0ab7e054 - 20
		{txID: "5", party: party, powTxID: "5B87F9DFA41DABE84A11CA78D9FE11DA8FC2AA926004CA66454A7AF0A206480D", powNonce: 4095356}, // 0000077b7d66117b57e45ccba0c31554e61c9853cc1cd9a2cf09c41b0aa9c22e - 21
		{txID: "6", party: party, powTxID: "B14DD602ED48C9F7B5367105A4A97FFC9199EA0C9E1490B786534768DD1538EF", powNonce: 1751582}, // 000003bbf0cde49e3899ad23282b18defbc12a65f07c95d768464b87024df368 - 22
		{txID: "7", party: party, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336},  // 000001c297318619efd60b9197f89e36fea83ca8d7461cf7b7c78af84e0a3b51 - 23
	}
	testBanWithTxPermutations(t, e, txs, false, 102, party)

	// test transactions not satisfying the required increasing difficulty leading to ban of the party
	txs = []*testTx{
		{txID: "8", party: party, powTxID: "2A1319636230740888C968E4E7610D6DE820E644EEC3C08AA5322A0A022014BD", powNonce: 1421231},  // 000009c5043c4e1dd7fe190ece8d3fd83d94c4e2a2b7800456ce5f5a653c9f75 - 20
		{txID: "9", party: party, powTxID: "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F", powNonce: 424517},   // 00000e31f8ac983354f5885d46b7631bc75f69ec82e8f6178bae53db0ab7e054 - 20
		{txID: "10", party: party, powTxID: "5B0E1EB96CCAC120E6D824A5F4C4007EABC59573B861BD84B1EF09DFB376DC84", powNonce: 4031737}, // 000002a98320df372412d7179ca2645b13ff3ecbe660e4a9a743fb423d8aec1f - 22
		{txID: "11", party: party, powTxID: "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0", powNonce: 431336},  // 000001c297318619efd60b9197f89e36fea83ca8d7461cf7b7c78af84e0a3b51 - 23
	}
	testBanWithTxPermutations(t, e, txs, true, 126, party)
}

func testBanWithTxPermutations(t *testing.T, e *Engine, txs []*testTx, expectedBan bool, blockHeight uint64, party string) {
	t.Helper()
	txsPerm := permutation(txs)
	for i, perm := range txsPerm {
		// clear any bans
		e.bannedParties = map[string]uint64{}

		// begin a new block
		e.BeginBlock(blockHeight+uint64(i), "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4")

		// send the transactions with the given permutation
		for _, p := range perm {
			p.blockHeight = blockHeight + uint64(i)
			require.NoError(t, e.DeliverTx(p))
		}

		// end the block to check if the party was playing nice
		e.EndOfBlock()

		// verify expected ban
		if expectedBan {
			require.Equal(t, 1, len(e.bannedParties))
			require.Equal(t, uint64(4), e.bannedParties[party])
			require.Equal(t, 0, len(e.blockPartyToPoW))
		} else {
			require.Equal(t, 0, len(e.bannedParties))
			require.Equal(t, 0, len(e.blockPartyToPoW))
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

func TestOnEpoch(t *testing.T) {
	e := New(logging.NewTestLogger(), NewDefaultConfig(), &TestEpochEngine{})
	e.UpdateSpamPoWNumberOfPastBlocks(context.Background(), num.NewUint(1))
	e.UpdateSpamPoWDifficulty(context.Background(), num.NewUint(20))
	e.UpdateSpamPoWHashFunction(context.Background(), crypto.Sha3)
	e.UpdateSpamPoWNumberOfTxPerBlock(context.Background(), num.NewUint(1))

	e.OnEpochEvent(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_UNSPECIFIED, Seq: 100})
	require.NotEqual(t, 100, e.currentEpoch)

	e.bannedParties["party1"] = 100

	e.OnEpochEvent(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, Seq: 100})
	require.NotEqual(t, 100, e.currentEpoch)

	require.Equal(t, 1, len(e.bannedParties))
	e.OnEpochEvent(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, Seq: 101})
	require.Equal(t, 0, len(e.bannedParties))
}

type testTx struct {
	party       string
	blockHeight uint64
	powNonce    uint64
	powTxID     string
	txID        string
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
