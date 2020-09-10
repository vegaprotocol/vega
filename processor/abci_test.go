package processor_test

import (
	"crypto"
	"encoding/hex"
	"errors"
	"log"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/processor"
	types "code.vegaprotocol.io/vega/proto"
	vegacrypto "code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/abci/types"
	tmprototypes "github.com/tendermint/tendermint/proto/types"
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
		proc.col,
		nil,
		proc.gov,
		proc.notary,
		proc.stat,
		proc.ts,
		proc.top,
		proc.wallet,
	)
}

func (s *AbciTestSuite) testProcessCommandSucess(t *testing.T, app *processor.App, proc *procTest) {
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	data := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		blockchain.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		// blockchain.AmendOrderCommand: &types.OrderAmendment{
		// 	PartyID: party,
		// },
		blockchain.ProposeCommand: &types.Proposal{
			PartyID: party,
			Terms:   &types.ProposalTerms{}, // avoid nil bit, shouldn't be asset
		},
		blockchain.VoteCommand: &types.Vote{
			PartyID: party,
		},
		// blockchain.WithdrawCommand: &types.Withdraw{
		// 	PartyID: party,
		// },
	}
	zero := uint64(0)

	// proc.stat.EXPECT().IncTotalAmendOrder().Times(1)
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
	proc.gov.EXPECT().SubmitProposal(gomock.Any(), gomock.Any()).Times(1).Return(nil)

	for cmd, msg := range data {
		pub, priv, err := s.sig.GenKey()
		require.NoError(t, err)

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
		log.Printf("resp = %+v\n", resp)
	}

}

func (s *AbciTestSuite) testBeginCommitSuccess(t *testing.T, app *processor.App, proc *procTest) {
	now := time.Now()
	prev := now.Add(-time.Second)

	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))

	proc.ts.EXPECT().SetTimeNow(gomock.Any(), now).Times(1)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	proc.cmd.EXPECT().Command(blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)
	app.OnBeginBlock(tmtypes.RequestBeginBlock{
		Header: tmprototypes.Header{
			Time: now,
		},
	})

	duration := time.Duration(now.UnixNano() - prev.UnixNano()).Seconds()
	var (
		totBatches        = uint64(1)
		zero       uint64 = 0
	)

	proc.eng.EXPECT().Generate().Times(1).Return(nil)
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

	app.OnCommit()
}

func (s *AbciTestSuite) testBeginRegisterError(t *testing.T, app *processor.App, proc *procTest) {
	now := time.Now()
	prev := now.Add(-time.Second)
	expErr := errors.New("test error")
	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.ts.EXPECT().SetTimeNow(gomock.Any(), now).Times(1)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	proc.cmd.EXPECT().Command(blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ blockchain.Command, payload proto.Message) {
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(expErr)

	app.OnBeginBlock(tmtypes.RequestBeginBlock{
		Header: tmprototypes.Header{
			Time: now,
		},
	})
}

func (s *AbciTestSuite) testBeginCallsCommanderOnce(t *testing.T, app *processor.App, proc *procTest) {
	now := time.Now()
	prev := now.Add(-time.Second)
	proc.top.EXPECT().Ready().AnyTimes().Return(false)
	proc.top.EXPECT().SelfChainPubKey().AnyTimes().Return([]byte("tmpubkey"))
	proc.ts.EXPECT().SetTimeNow(gomock.Any(), gomock.Any()).Times(2)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	proc.cmd.EXPECT().Command(blockchain.RegisterNodeCommand, gomock.Any()).Times(1).Do(func(_ blockchain.Command, payload proto.Message) {
		// check if the type is ok
		_, ok := payload.(*types.NodeRegistration)
		assert.True(t, ok)
	}).Return(nil)
	app.OnBeginBlock(tmtypes.RequestBeginBlock{
		Header: tmprototypes.Header{
			Time: now,
		},
	})

	// next block times
	prev, now = now, now.Add(time.Second)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev, nil)
	app.OnBeginBlock(tmtypes.RequestBeginBlock{
		Header: tmprototypes.Header{
			Time: now,
		},
	})
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
		{"Call Begin, register node error - fail", s.testBeginRegisterError},
		{"Call Begin twice, only calls commander once", s.testBeginCallsCommanderOnce},
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
