package processor_test

import (
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/processor/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type procTest struct {
	*processor.Processor
	eng         *mocks.MockExecutionEngine
	ts          *mocks.MockTimeService
	stat        *mocks.MockStats
	tickCB      func(time.Time)
	ctrl        *gomock.Controller
	cmd         *mocks.MockCommander
	wallet      *mocks.MockWallet
	assets      *mocks.MockAssets
	top         *mocks.MockValidatorTopology
	gov         *mocks.MockGovernanceEngine
	proposalBuf *mocks.MockProposalBuf
}

type stubWallet struct {
	key    []byte
	chain  string
	signed []byte
	err    error
}

func getTestProcessor(t *testing.T) *procTest {
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	eng := mocks.NewMockExecutionEngine(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	stat := mocks.NewMockStats(ctrl)
	cmd := mocks.NewMockCommander(ctrl)
	wallet := mocks.NewMockWallet(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	top := mocks.NewMockValidatorTopology(ctrl)
	gov := mocks.NewMockGovernanceEngine(ctrl)
	proposalBuf := mocks.NewMockProposalBuf(ctrl)

	//top.EXPECT().Ready().AnyTimes().Return(true)
	var cb func(time.Time)
	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1).Do(func(c func(time.Time)) {
		cb = c
	})
	wal := getTestStubWallet()
	wallet.EXPECT().Get(nodewallet.Vega).Times(1).Return(wal, true)

	proc, err := processor.New(log, processor.NewDefaultConfig(), eng, ts, stat, cmd, wallet, assets, top, gov, proposalBuf, true)
	assert.NoError(t, err)
	return &procTest{
		Processor: proc,
		eng:       eng,
		ts:        ts,
		stat:      stat,
		tickCB:    cb,
		ctrl:      ctrl,
		cmd:       cmd,
		wallet:    wallet,
		assets:    assets,
		top:       top,
	}
}

func getTestStubWallet() *stubWallet {
	return &stubWallet{
		key:   []byte("test key"),
		chain: string(nodewallet.Vega),
	}
}

func TestValidateSigned(t *testing.T) {
	t.Run("Test all signed commands basic - success", testValidateCommandsSuccess)
	t.Run("Test all signed commands basic - failure", testValidateCommandsFail)
	t.Run("Test submit order validations - success", testSubmitOrderValidationSuccess)
	t.Run("Test submit order validation - failure", testSubmitOrderValidationFail)
	t.Run("Test validate signed invalid payload", testValidateSignedInvalidPayload)
	t.Run("Test validate signed - invalid command", testValidateSignedInvalidCommand)
}

func TestProcess(t *testing.T) {
	t.Run("Test all basic process commands - Success", testProcessCommandSuccess)
	t.Run("Test process asset proposal - Success", testProcessAssetProposalSuccess)
}

func TestBeginCommit(t *testing.T) {
	t.Run("Call Begin and Commit - success", testBeginCommitSuccess)
	t.Run("Call begin, register node error - fail", testBeginRegisterError)
	t.Run("Call Begin twice, only calls commander once", testBeginCallsCommanderOnce)
}

func TestOnTick(t *testing.T) {
	t.Run("Test onTick callback with empty data-set", testOnTickEmpty)
	t.Run("Test onTick callback with pending data", testOnTickPending)
	t.Run("Test onTick callback with validated proposal", testOnTickSubmit)
	t.Run("Test onTick callback with validated proposal, retry", testOnTickSubmitRetry)
	t.Run("Test onTick callback with partially validated proposal (rejected)", testOnTickReject)
	t.Run("Test onTick complex: actual node votes + retry", testOnTickWithNodes)
}

func testOnTickEmpty(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	// this is to simulate what happens on timer tick when there aren't any proposals
	proc.tickCB(time.Now())
}

func testOnTickPending(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	// first, Begin a block (sets up the timestamps)
	now := time.Now()
	prev := now.Add(-time.Second)
	next := now.Add(time.Second) // 1 second later

	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().Len().AnyTimes().Return(1)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.top.EXPECT().AddNodeRegistration(gomock.Any()).AnyTimes().Return(nil)

	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	// Begin was never called, so we expect nodewallet to be involved
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)

	assetID := "asset-id-1"
	asset := assetStub{
		valid: true,
		err:   nil,
	}
	proc.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any()).Times(1).Return(assetID, nil)
	proc.assets.EXPECT().Get(gomock.Any()).Times(1).Return(asset, nil)

	// call Begin, expect no error
	assert.NoError(t, proc.Begin())
	// submit an asset proposal
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	closeTS := time.Now().Add(120 * time.Hour)
	validTS := time.Now().Add(24 * time.Hour)
	data := &types.Proposal{
		PartyID:   party,
		Reference: "proposal-ref",
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetSource{
						Source: &types.AssetSource_BuiltinAsset{
							BuiltinAsset: &types.BuiltinAsset{},
						},
					},
				},
			},
			ClosingTimestamp:    closeTS.Unix(),
			ValidationTimestamp: validTS.Unix(),
		},
	}
	payload, err := proto.Marshal(data)
	assert.NoError(t, err)
	ch := make(chan struct{}, 1)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.NodeVoteCommand, gomock.Any()).Times(1).Return(nil).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		nv, ok := payload.(*types.NodeVote)
		// make sure the correct command was issued
		assert.True(t, ok)
		assert.Equal(t, data.Reference, nv.Reference)
		ch <- struct{}{}
	})
	assert.NoError(t, proc.Process(payload, blockchain.ProposeCommand))
	// Register a node, so the proposal is still pending
	wal := getTestStubWallet()
	reg := &types.NodeRegistration{
		PubKey: wal.key,
	}
	payload, err = proto.Marshal(reg)
	assert.NoError(t, proc.Process(payload, blockchain.RegisterNodeCommand))

	// next time tick, proposal is pending but not past validation time
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if len(ch) > 0 {
			break
		}
		select {
		case _ = <-tick.C:
			proc.tickCB(next)
			next = next.Add(1 * time.Second)
		}
	}
}

func testOnTickSubmit(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	// first, Begin a block (sets up the timestamps)
	now := time.Now()
	prev := now.Add(-time.Second)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)

	assetID := "asset-id-1"
	asset := assetStub{
		valid: true,
		err:   nil,
	}

	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(true)
	proc.top.EXPECT().Len().AnyTimes().Return(1)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.top.EXPECT().AddNodeRegistration(gomock.Any()).AnyTimes().Return(nil)

	proc.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any()).Times(1).Return(assetID, nil)
	proc.assets.EXPECT().Get(gomock.Any()).Times(1).Return(asset, nil)

	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)
	// call Begin, expect no error
	assert.NoError(t, proc.Begin())
	// submit an asset proposal
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	closeTS := time.Now().Add(120 * time.Hour)
	validTS := time.Now().Add(24 * time.Hour)
	data := &types.Proposal{
		PartyID:   party,
		Reference: "proposal-ref",
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetSource{
						Source: &types.AssetSource_BuiltinAsset{
							BuiltinAsset: &types.BuiltinAsset{},
						},
					},
				},
			},
			ClosingTimestamp:    closeTS.Unix(),
			ValidationTimestamp: validTS.Unix(),
		},
	}
	payload, err := proto.Marshal(data)
	assert.NoError(t, err)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.NodeVoteCommand, gomock.Any()).Times(1).Return(nil).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		nv, ok := payload.(*types.NodeVote)
		// make sure the correct command was issued
		assert.True(t, ok)
		assert.Equal(t, data.Reference, nv.Reference)
	})
	assert.NoError(t, proc.Process(payload, blockchain.ProposeCommand))
	wal := getTestStubWallet()
	vote := &types.NodeVote{
		PubKey:    wal.PubKeyOrAddress(),
		Reference: data.Reference,
	}
	payload, err = proto.Marshal(vote)
	assert.NoError(t, proc.Process(payload, blockchain.NodeVoteCommand))

	ch := make(chan struct{}, 1)
	proc.eng.EXPECT().SubmitProposal(gomock.Any()).Times(1).Return(nil).Do(func(sp *types.Proposal) {
		assert.Equal(t, data.Reference, sp.Reference)
		assert.Equal(t, data.PartyID, sp.PartyID)
		ch <- struct{}{}
	})
	// next time tick, proposal is pending but not past validation time

	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if len(ch) > 0 {
			break
		}
		select {
		case _ = <-tick.C:
			proc.tickCB(validTS.Add(time.Second))
			validTS = validTS.Add(time.Second)
		}
	}

}

func testOnTickSubmitRetry(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	// first, Begin a block (sets up the timestamps)
	now := time.Now()
	prev := now.Add(-time.Second)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	assetID := "asset-id-1"
	asset := assetStub{
		valid: true,
		err:   nil,
	}

	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(true)
	proc.top.EXPECT().Len().AnyTimes().Return(1)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.top.EXPECT().AddNodeRegistration(gomock.Any()).AnyTimes().Return(nil)

	proc.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any()).Times(1).Return(assetID, nil)
	proc.assets.EXPECT().Get(gomock.Any()).Times(1).Return(asset, nil)

	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)
	// call Begin, expect no error
	assert.NoError(t, proc.Begin())
	// submit an asset proposal
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	closeTS := time.Now().Add(120 * time.Hour)
	validTS := time.Now().Add(24 * time.Hour)
	data := &types.Proposal{
		PartyID:   party,
		Reference: "proposal-ref",
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetSource{
						Source: &types.AssetSource_BuiltinAsset{
							BuiltinAsset: &types.BuiltinAsset{},
						},
					},
				},
			},
			ClosingTimestamp:    closeTS.Unix(),
			ValidationTimestamp: validTS.Unix(),
		},
	}
	payload, err := proto.Marshal(data)
	assert.NoError(t, err)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.NodeVoteCommand, gomock.Any()).Times(1).Return(nil).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		nv, ok := payload.(*types.NodeVote)
		// make sure the correct command was issued
		assert.True(t, ok)
		assert.Equal(t, data.Reference, nv.Reference)
	})
	assert.NoError(t, proc.Process(payload, blockchain.ProposeCommand))

	wal := getTestStubWallet()
	vote := &types.NodeVote{
		PubKey:    wal.PubKeyOrAddress(),
		Reference: data.Reference,
	}
	payload, err = proto.Marshal(vote)
	assert.NoError(t, proc.Process(payload, blockchain.NodeVoteCommand))

	i := 0
	returns := []error{
		errors.New("random error for first call"),
		nil,
	}
	ch := make(chan struct{}, 1)
	proc.eng.EXPECT().SubmitProposal(gomock.Any()).Times(2).DoAndReturn(func(sp *types.Proposal) error {
		assert.Equal(t, data.Reference, sp.Reference)
		assert.Equal(t, data.PartyID, sp.PartyID)
		ret := returns[i]
		if ret == nil {
			ch <- struct{}{}
		}
		i++
		return ret
	})
	// next block - set timestamps
	next := validTS.Add(time.Second)
	// next time tick, proposal is pending but not past validation time
	//proc.tickCB(next) // this submit should fail
	// next tick, we ought to try again
	//proc.tickCB(next.Add(time.Second))
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if len(ch) > 0 {
			break
		}
		select {
		case _ = <-tick.C:
			proc.tickCB(next)
			next = next.Add(time.Second)
		}
	}

}

func testOnTickWithNodes(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	// first, Begin a block (sets up the timestamps)
	now := time.Now()
	prev := now.Add(-time.Second)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)

	assetID := "asset-id-1"
	asset := assetStub{
		valid: true,
		err:   nil,
	}
	proc.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any()).Times(1).Return(assetID, nil)
	proc.assets.EXPECT().Get(gomock.Any()).Times(1).Return(asset, nil)

	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(true)
	proc.top.EXPECT().Len().AnyTimes().Return(1)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.top.EXPECT().AddNodeRegistration(gomock.Any()).AnyTimes().Return(nil)

	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)
	// call Begin, expect no error
	assert.NoError(t, proc.Begin())
	// submit an asset proposal
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	closeTS := time.Now().Add(120 * time.Hour)
	validTS := time.Now().Add(24 * time.Hour)
	data := &types.Proposal{
		PartyID:   party,
		Reference: "proposal-ref",
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetSource{
						Source: &types.AssetSource_BuiltinAsset{
							BuiltinAsset: &types.BuiltinAsset{},
						},
					},
				},
			},
			ClosingTimestamp:    closeTS.Unix(),
			ValidationTimestamp: validTS.Unix(),
		},
	}
	payload, err := proto.Marshal(data)
	assert.NoError(t, err)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.NodeVoteCommand, gomock.Any()).Times(1).Return(nil).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		nv, ok := payload.(*types.NodeVote)
		// make sure the correct command was issued
		assert.True(t, ok)
		assert.Equal(t, data.Reference, nv.Reference)
	})
	assert.NoError(t, proc.Process(payload, blockchain.ProposeCommand))

	// This node has received the proposal and validated it. We want to vote for it
	// first process the RegisterNodeCommand transaction
	prev, now = now, now.Add(time.Second)
	wal := getTestStubWallet()
	reg := &types.NodeRegistration{
		PubKey: wal.key,
	}
	payload, err = proto.Marshal(reg)
	assert.NoError(t, err)
	assert.NoError(t, proc.Process(payload, blockchain.RegisterNodeCommand))

	// Now this node can vote has to vote on the proposal
	vote := &types.NodeVote{
		PubKey:    wal.PubKeyOrAddress(),
		Reference: data.Reference,
	}
	payload, err = proto.Marshal(vote)
	assert.NoError(t, proc.Process(payload, blockchain.NodeVoteCommand))

	i := 0
	returns := []error{
		errors.New("random error for first call"),
		nil,
	}
	ch := make(chan struct{}, 1)
	proc.eng.EXPECT().SubmitProposal(gomock.Any()).Times(2).DoAndReturn(func(sp *types.Proposal) error {
		assert.Equal(t, data.Reference, sp.Reference)
		assert.Equal(t, data.PartyID, sp.PartyID)
		ret := returns[i]
		i++
		if ret == nil {
			ch <- struct{}{}
		}
		return ret
	})
	// next block - set timestamps
	next := validTS.Add(time.Second)
	// next time tick, proposal is pending but not past validation time
	// proc.tickCB(next) // this submit should fail
	// next tick, we ought to try again
	// proc.tickCB(next.Add(time.Second))

	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if len(ch) > 0 {
			break
		}
		select {
		case _ = <-tick.C:
			proc.tickCB(next)
			next = next.Add(1 * time.Second)
		}
	}

}

func testOnTickReject(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	// first, Begin a block (sets up the timestamps)
	now := time.Now()
	prev := now.Add(-time.Second)
	next := now.Add(time.Hour * 96) //  4 days later, the validation period has expired for sure

	assetID := "asset-id-1"
	asset := assetStub{
		valid: true,
		err:   nil,
	}
	proc.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any()).Times(1).Return(assetID, nil)
	proc.assets.EXPECT().Get(gomock.Any()).Times(1).Return(asset, nil)

	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(true)
	proc.top.EXPECT().Len().AnyTimes().Return(1)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.top.EXPECT().AddNodeRegistration(gomock.Any()).AnyTimes().Return(nil)

	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)
	// call Begin, expect no error
	assert.NoError(t, proc.Begin())

	// receive registration command of another node
	secondNode := []byte("node-key")
	nr := &types.NodeRegistration{
		PubKey: secondNode,
	}
	reg, err := proto.Marshal(nr)
	assert.NoError(t, err)
	assert.NoError(t, proc.Process(reg, blockchain.RegisterNodeCommand))

	closeTS := time.Now().Add(120 * time.Hour)
	validTS := time.Now().Add(24 * time.Hour)
	// submit an asset proposal
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	data := &types.Proposal{
		PartyID:   party,
		Reference: "proposal-ref",
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetSource{
						Source: &types.AssetSource_BuiltinAsset{
							BuiltinAsset: &types.BuiltinAsset{},
						},
					},
				},
			},
			ClosingTimestamp:    closeTS.Unix(),
			ValidationTimestamp: validTS.Unix(),
		},
	}
	payload, err := proto.Marshal(data)
	assert.NoError(t, err)
	ch := make(chan struct{}, 1)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.NodeVoteCommand, gomock.Any()).Times(1).Return(nil).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		nv, ok := payload.(*types.NodeVote)
		// make sure the correct command was issued
		assert.True(t, ok)
		assert.Equal(t, data.Reference, nv.Reference)
		ch <- struct{}{}
	})
	assert.NoError(t, proc.Process(payload, blockchain.ProposeCommand))

	// We expect SubmitProposal to NOT be called (other node did NOT validate
	proc.eng.EXPECT().SubmitProposal(gomock.Any()).Times(0).Return(nil)

	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if len(ch) > 0 {
			break
		}
		select {
		case _ = <-tick.C:
			proc.tickCB(next)
			next = next.Add(1 * time.Second)
		}
	}

}

func testBeginCommitSuccess(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	var zero uint64 = 0
	totBatches := uint64(1)
	now := time.Now()
	prev := now.Add(-time.Second)
	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))

	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)
	// call Begin, expect no error
	assert.NoError(t, proc.Begin())
	proc.eng.EXPECT().Generate().Times(1).Return(nil)
	duration := time.Duration(now.UnixNano() - prev.UnixNano()).Seconds()
	proc.stat.EXPECT().SetBlockDuration(uint64(duration * float64(time.Second.Nanoseconds()))).Times(1)
	proc.stat.EXPECT().IncTotalBatches().Times(1).Do(func() {
		totBatches++
	})
	proc.stat.EXPECT().TotalOrders().Times(1).Return(zero)
	proc.stat.EXPECT().TotalBatches().Times(2).DoAndReturn(func() uint64 {
		return totBatches
	})
	proc.stat.EXPECT().SetAverageOrdersPerBatch(zero).Times(1)
	proc.stat.EXPECT().CurrentOrdersInBatch().Times(2).Return(zero)
	proc.stat.EXPECT().CurrentTradesInBatch().Times(2).Return(zero)
	proc.stat.EXPECT().SetOrdersPerSecond(zero).Times(1)
	proc.stat.EXPECT().SetTradesPerSecond(zero).Times(1)
	proc.stat.EXPECT().NewBatch().Times(1)
	assert.NoError(t, proc.Commit())
}

func testBeginRegisterError(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	now := time.Now()
	prev := now.Add(-time.Second)
	expErr := errors.New("test error")
	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(expErr)
	err := proc.Begin()
	assert.Error(t, err)
	assert.Equal(t, expErr, err)
}

func testBeginCallsCommanderOnce(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()

	now := time.Now()
	prev := now.Add(-time.Second)
	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)
	assert.NoError(t, proc.Begin())
	// next block times
	prev, now = now, now.Add(time.Second)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	assert.NoError(t, proc.Begin())
}

func testValidateSignedInvalidPayload(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	party := []byte("party-id")
	cmd := blockchain.VoteCommand
	// wrong type for this command
	payload, err := proto.Marshal(&types.Proposal{
		PartyID: hex.EncodeToString(party),
	})
	assert.NoError(t, err)
	err = proc.ValidateSigned(party, payload, cmd)
	assert.Error(t, err)
}

func testValidateSignedInvalidCommand(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	var b byte // nil value
	assert.Error(t, proc.ValidateSigned([]byte("party"), []byte("foobar"), blockchain.Command(b)))
}

func testValidateCommandsFail(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString([]byte("another-party"))
	data := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		blockchain.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		blockchain.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		blockchain.ProposeCommand: &types.Proposal{
			PartyID: party,
		},
		blockchain.VoteCommand: &types.Vote{
			PartyID: party,
		},
		blockchain.WithdrawCommand: &types.Withdraw{
			PartyID: party,
		},
	}
	expError := map[blockchain.Command]error{
		blockchain.SubmitOrderCommand: processor.ErrOrderSubmissionPartyAndPubKeyDoesNotMatch,
		blockchain.CancelOrderCommand: processor.ErrOrderCancellationPartyAndPubKeyDoesNotMatch,
		blockchain.AmendOrderCommand:  processor.ErrOrderAmendmentPartyAndPubKeyDoesNotMatch,
		blockchain.ProposeCommand:     processor.ErrProposalSubmissionPartyAndPubKeyDoesNotMatch,
		blockchain.VoteCommand:        processor.ErrVoteSubmissionPartyAndPubKeyDoesNotMatch,
		blockchain.WithdrawCommand:    processor.ErrWithdrawPartyAndPublKeyDoesNotMatch,
	}
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	for cmd, msg := range data {
		payload, err := proto.Marshal(msg)
		assert.NoError(t, err)
		err = proc.ValidateSigned(key, payload, cmd)
		assert.Error(t, err)
		expErr, ok := expError[cmd]
		assert.True(t, ok)
		assert.Equal(t, expErr, err)
	}
}

func testValidateCommandsSuccess(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	data := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		blockchain.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		blockchain.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		blockchain.VoteCommand: &types.Vote{
			PartyID: party,
		},
		blockchain.WithdrawCommand: &types.Withdraw{
			PartyID: party,
		},
		blockchain.ProposeCommand: &types.Proposal{
			PartyID: party,
		},
	}
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	for cmd, msg := range data {
		payload, err := proto.Marshal(msg)
		assert.NoError(t, err)
		assert.NoError(t, proc.ValidateSigned(key, payload, cmd), "Failed to validate %v command payload", cmd)
	}
}

func testSubmitOrderValidationSuccess(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	party := []byte("party-id")
	// bare bones
	sub := &types.OrderSubmission{
		MarketID: "market-id",
		PartyID:  hex.EncodeToString(party),
		Price:    1,
		Size:     1,
	}
	payload, err := proto.Marshal(sub)
	assert.NoError(t, err)
	assert.NoError(t, proc.ValidateSigned(party, payload, blockchain.SubmitOrderCommand))
}

func testSubmitOrderValidationFail(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	// different party
	party := []byte("other-party")
	sub := &types.OrderSubmission{
		MarketID: "market-id",
		PartyID:  hex.EncodeToString([]byte("party-id")),
		Price:    1,
		Size:     1,
	}
	payload, err := proto.Marshal(sub)
	assert.NoError(t, err)
	err = proc.ValidateSigned(party, payload, blockchain.SubmitOrderCommand)
	assert.Error(t, err)
	assert.Equal(t, err, processor.ErrOrderSubmissionPartyAndPubKeyDoesNotMatch)
}

func testProcessCommandSuccess(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	data := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		blockchain.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		blockchain.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		blockchain.ProposeCommand: &types.Proposal{
			PartyID: party,
			Terms:   &types.ProposalTerms{}, // avoid nil bit, shouldn't be asset
		},
		blockchain.VoteCommand: &types.Vote{
			PartyID: party,
		},
		blockchain.WithdrawCommand: &types.Withdraw{
			PartyID: party,
		},
		blockchain.NotifyTraderAccountCommand: &types.NotifyTraderAccount{
			TraderID: party,
		},
	}
	zero := uint64(0)
	proc := getTestProcessor(t)
	proc.stat.EXPECT().IncTotalAmendOrder().Times(1)
	proc.stat.EXPECT().IncTotalCancelOrder().Times(1)
	proc.stat.EXPECT().IncTotalCreateOrder().Times(1)
	// creating an order, should be no trades
	proc.stat.EXPECT().IncTotalOrders().Times(1)
	proc.stat.EXPECT().AddCurrentTradesInBatch(zero).Times(1)
	proc.stat.EXPECT().AddTotalTrades(zero).Times(1)
	proc.stat.EXPECT().IncCurrentOrdersInBatch().Times(1)

	proc.eng.EXPECT().Withdraw(gomock.Any()).Times(1).Return(nil)
	proc.eng.EXPECT().SubmitOrder(gomock.Any()).Times(1).Return(&types.OrderConfirmation{}, nil)
	proc.eng.EXPECT().CancelOrder(gomock.Any()).Times(1).Return(&types.OrderCancellationConfirmation{}, nil)
	proc.eng.EXPECT().AmendOrder(gomock.Any()).Times(1).Return(&types.OrderConfirmation{}, nil)
	proc.eng.EXPECT().VoteOnProposal(gomock.Any()).Times(1).Return(nil)
	proc.eng.EXPECT().SubmitProposal(gomock.Any()).Times(1).Return(nil)
	proc.eng.EXPECT().NotifyTraderAccount(gomock.Any()).Times(1).Return(nil)
	defer proc.ctrl.Finish()
	for cmd, msg := range data {
		payload, err := proto.Marshal(msg)
		assert.NoError(t, err)
		assert.NoError(t, proc.Process(payload, cmd), "Failed to process %v command payload", cmd)
	}
}

func testProcessAssetProposalSuccess(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	// set current timetamps
	now := time.Now()
	prev := now.Add(-time.Second)
	next := now.Add(time.Second)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	assetID := "asset-id-1"
	asset := assetStub{
		valid: true,
		err:   nil,
	}

	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.top.EXPECT().Len().AnyTimes().Return(1)
	proc.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any()).Times(1).Return(assetID, nil)
	proc.assets.EXPECT().Get(gomock.Any()).Times(1).Return(asset, nil)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)
	// call Begin, expect no error
	assert.NoError(t, proc.Begin())
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	closeTS := time.Now().Add(120 * time.Hour)
	validTS := time.Now().Add(2 * time.Hour)
	data := &types.Proposal{
		PartyID:   party,
		Reference: "proposal-ref",
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetSource{
						Source: &types.AssetSource_BuiltinAsset{
							BuiltinAsset: &types.BuiltinAsset{},
						},
					},
				},
			},
			ClosingTimestamp:    closeTS.Unix(),
			ValidationTimestamp: validTS.Unix(),
		},
	}
	payload, err := proto.Marshal(data)
	assert.NoError(t, err)
	ch := make(chan struct{}, 1)
	proc.cmd.EXPECT().Command(gomock.Any(), blockchain.NodeVoteCommand, gomock.Any()).Times(1).Return(nil).Do(func(_ nodewallet.Wallet, _ blockchain.Command, payload proto.Message) {
		nv, ok := payload.(*types.NodeVote)
		// make sure the correct command was issued
		assert.True(t, ok)
		assert.Equal(t, data.Reference, nv.Reference)
		// notify the call happend
		ch <- struct{}{}
	})
	assert.NoError(t, proc.Process(payload, blockchain.ProposeCommand))

	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if len(ch) > 0 {
			break
		}
		select {
		case _ = <-tick.C:
			proc.tickCB(next)
			next = next.Add(1 * time.Second)
		}
	}
}

func (s stubWallet) Chain() string {
	return s.chain
}

func (s stubWallet) PubKeyOrAddress() []byte {
	return s.key
}

func (s stubWallet) Sign(_ []byte) ([]byte, error) {
	return s.signed, s.err
}

type assetStub struct {
	valid bool
	err   error
}

func (a assetStub) Data() *types.Asset                      { return nil }
func (a assetStub) GetAssetClass() common.AssetClass        { return common.ERC20 }
func (a assetStub) IsValid() bool                           { return a.valid }
func (a assetStub) Validate() error                         { return a.err }
func (a assetStub) SignBridgeWhitelisting() ([]byte, error) { return nil, nil }
func (a assetStub) ValidateWithdrawal() error               { return nil }
func (a assetStub) SignWithdrawal() ([]byte, error)         { return nil, nil }
func (a assetStub) ValidateDeposit() error                  { return nil }
func (a assetStub) String() string                          { return "" }
