package processor_test

import (
	"context"
	"crypto"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/processor"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/txn"
	vegacrypto "code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

type AbciTestSuite struct {
	sig *vegacrypto.SignatureAlgorithm
}

func (s *AbciTestSuite) signedTx(t *testing.T, tx *types.Transaction, key crypto.PrivateKey) *types.SignedBundle {
	txBytes, err := proto.Marshal(tx)
	require.NoError(t, err)

	sig, err := s.sig.Sign(key, txBytes)
	require.NoError(t, err)

	stx := &types.SignedBundle{
		Tx: txBytes,
		Sig: &types.Signature{
			Algo: s.sig.Name(),
			Sig:  sig,
		},
	}

	return stx
}

func (s *AbciTestSuite) newApp(proc *procTest) (*processor.App, error) {
	return processor.NewApp(
		logging.NewTestLogger(),
		processor.NewDefaultConfig(),
		nil,
		proc.assets,
		proc.bank,
		nil, // broker
		proc.erc,
		proc.evtfwd,
		proc.eng,
		proc.cmd,
		nil,
		proc.gov,
		proc.notary,
		proc.stat,
		proc.ts,
		proc.top,
		proc.wallet,
		proc.netp,
	)
}

func (s *AbciTestSuite) testProcessCommandSucess(t *testing.T, app *processor.App, proc *procTest) {
	pub, priv, err := s.sig.GenKey()
	require.NoError(t, err)

	party := hex.EncodeToString(pub.([]byte))
	data := map[txn.Command]proto.Message{
		txn.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		txn.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		// txn.AmendOrderCommand: &types.OrderAmendment{
		// 	PartyID: party,
		// },
		txn.ProposeCommand: &types.Proposal{
			PartyID: party,
			Terms:   &types.ProposalTerms{}, // avoid nil bit, shouldn't be asset
		},
		txn.VoteCommand: &types.Vote{
			PartyID: party,
		},
		// txn.WithdrawCommand: &types.Withdraw{
		// 	PartyID: party,
		// },
	}
	zero := uint64(0)

	// proc.stat.EXPECT().IncTotalAmendOrder().Times(1)
	proc.stat.EXPECT().IncTotalTxCurrentBatch().AnyTimes()
	proc.stat.EXPECT().Height().AnyTimes()
	proc.stat.EXPECT().SetAverageTxSizeBytes(gomock.Any()).AnyTimes()
	proc.stat.EXPECT().IncTotalTxCurrentBatch().AnyTimes()

	proc.stat.EXPECT().IncTotalCancelOrder().Times(1)
	proc.stat.EXPECT().IncTotalCreateOrder().Times(1)
	// creating an order, should be no trades
	proc.stat.EXPECT().IncTotalOrders().Times(1)
	proc.stat.EXPECT().AddCurrentTradesInBatch(zero).Times(1)
	proc.stat.EXPECT().AddTotalTrades(zero).Times(1)
	proc.stat.EXPECT().IncCurrentOrdersInBatch().Times(1)

	proc.eng.EXPECT().SubmitOrder(gomock.Any(), gomock.Any()).Times(1).Return(&types.OrderConfirmation{
		Order: &types.Order{},
	}, nil)
	proc.eng.EXPECT().CancelOrder(gomock.Any(), gomock.Any()).Times(1).Return([]*types.OrderCancellationConfirmation{}, nil)
	// proc.eng.EXPECT().AmendOrder(gomock.Any(), gomock.Any()).Times(1).Return(&types.OrderConfirmation{}, nil)
	proc.gov.EXPECT().AddVote(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	proc.gov.EXPECT().SubmitProposal(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)

	for cmd, msg := range data {
		tx := txEncode(t, cmd, msg)
		tx.From = &types.Transaction_PubKey{
			PubKey: pub.([]byte),
		}

		stx := s.signedTx(t, tx, priv)
		bz, err := proto.Marshal(stx)
		require.NoError(t, err)

		req := tmtypes.RequestDeliverTx{
			Tx: bz,
		}

		resp := app.Abci().DeliverTx(req)
		require.True(t, resp.IsOK())
	}

}

func (s *AbciTestSuite) testBeginCommitSuccess(t *testing.T, app *processor.App, proc *procTest) {
	now := time.Now()
	prev := now.Add(-time.Second)

	// stats
	proc.stat.EXPECT().SetAverageTxPerBatch(gomock.Any())
	proc.stat.EXPECT().SetTotalTxCurrentBatch(gomock.Any())
	proc.stat.EXPECT().SetTotalTxLastBatch(gomock.Any())
	proc.stat.EXPECT().TotalTxCurrentBatch()
	proc.stat.EXPECT().TotalTxLastBatch()
	proc.stat.EXPECT().IncHeight()

	proc.ts.EXPECT().SetTimeNow(gomock.Any(), now).Times(1)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	app.OnBeginBlock(tmtypes.RequestBeginBlock{
		Header: tmtypes.Header{
			Time: now,
		},
	})

	duration := time.Duration(now.UnixNano() - prev.UnixNano()).Seconds()
	var (
		totBatches        = uint64(1)
		zero       uint64 = 0
	)

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
	proc.eng.EXPECT().Hash().Times(1)

	app.OnCommit()
}

func (s *AbciTestSuite) testBeginCallsCommanderOnce(t *testing.T, app *processor.App, proc *procTest) {
	now := time.Now()
	prev := now.Add(-time.Second)
	proc.ts.EXPECT().SetTimeNow(gomock.Any(), gomock.Any()).Times(2)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	app.OnBeginBlock(tmtypes.RequestBeginBlock{
		Header: tmtypes.Header{
			Time: now,
		},
	})

	// next block times
	prev, now = now, now.Add(time.Second)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	app.OnBeginBlock(tmtypes.RequestBeginBlock{
		Header: tmtypes.Header{
			Time: now,
		},
	})
}

func (s *AbciTestSuite) testOnCheckTxFailWithNoBalances(t *testing.T, app *processor.App, proc *procTest) {
	proc.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(false)
	proc.bank.EXPECT().HasBalance(gomock.Any()).AnyTimes().Return(false)

	tx := txStub{pubkey: []byte("some pubkey")}
	_, resp := app.OnCheckTx(context.Background(), tmtypes.RequestCheckTx{}, tx)
	assert.Equal(t, resp.Code, uint32(51))
}

func (s *AbciTestSuite) testOnCheckTxSuccessWithBalance(t *testing.T, app *processor.App, proc *procTest) {
	proc.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(false)
	proc.bank.EXPECT().HasBalance(gomock.Any()).AnyTimes().Return(true)

	tx := txStub{pubkey: []byte("some pubkey")}
	_, resp := app.OnCheckTx(context.Background(), tmtypes.RequestCheckTx{}, tx)
	assert.Equal(t, resp.Code, uint32(0))
}

func TestAbci(t *testing.T) {
	sig := vegacrypto.NewEd25519()
	s := &AbciTestSuite{
		sig: &sig,
	}

	tests := []struct {
		name string
		fn   func(t *testing.T, app *processor.App, proc *procTest)
	}{
		{"Test all basic process commands - Success", s.testProcessCommandSucess},

		{"Call Begin and Commit - success", s.testBeginCommitSuccess},
		{"Call Begin twice, only calls commander once", s.testBeginCallsCommanderOnce},
		{"OnCheckTx fail with no balance", s.testOnCheckTxFailWithNoBalances},
		{"OnCheckTx success with balance", s.testOnCheckTxSuccessWithBalance},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			proc := getTestProcessor(t)
			defer proc.ctrl.Finish()

			app, err := s.newApp(proc)
			require.NoError(t, err)

			test.fn(t, app, proc)
		})
	}
}

type txStub struct {
	pubkey []byte
}

func (txStub) Command() txn.Command        { return txn.SubmitOrderCommand }
func (txStub) Unmarshal(interface{}) error { return nil }
func (t txStub) PubKey() []byte            { return t.pubkey }
func (txStub) Hash() []byte                { return nil }
func (txStub) Signature() []byte           { return nil }
func (txStub) Validate() error             { return nil }
func (txStub) BlockHeight() uint64         { return 0 }
