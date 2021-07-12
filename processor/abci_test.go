package processor_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/txn"
	vegacrypto "code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

type AbciTestSuite struct {
	sig *vegacrypto.SignatureAlgorithm
}

func (s *AbciTestSuite) newApp(proc *procTest) *processor.App {
	return processor.NewApp(
		logging.NewTestLogger(),
		processor.NewDefaultConfig(),
		nil,
		proc.assets,
		proc.bank,
		nil, // broker
		proc.witness,
		proc.evtfwd,
		proc.eng,
		proc.cmd,
		nil,
		proc.gov,
		proc.notary,
		proc.stat,
		proc.ts,
		proc.top,
		proc.netp,
		&processor.Oracle{
			Engine:   proc.oracles.Engine,
			Adaptors: proc.oracles.Adaptors,
		},
	)
}

func (s *AbciTestSuite) testBeginCommitSuccess(_ *testing.T, app *processor.App, proc *procTest) {
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
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev)
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

func (s *AbciTestSuite) testBeginCallsCommanderOnce(_ *testing.T, app *processor.App, proc *procTest) {
	now := time.Now()
	prev := now.Add(-time.Second)
	proc.ts.EXPECT().SetTimeNow(gomock.Any(), gomock.Any()).Times(2)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev)
	app.OnBeginBlock(tmtypes.RequestBeginBlock{
		Header: tmtypes.Header{
			Time: now,
		},
	})

	// next block times
	prev, now = now, now.Add(time.Second)
	proc.ts.EXPECT().GetTimeNow().Times(1).Return(now)
	proc.ts.EXPECT().GetTimeLastBatch().Times(1).Return(prev)
	app.OnBeginBlock(tmtypes.RequestBeginBlock{
		Header: tmtypes.Header{
			Time: now,
		},
	})
}

func (s *AbciTestSuite) testOnCheckTxFailWithNoBalances(t *testing.T, app *processor.App, proc *procTest) {
	proc.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(false)
	proc.bank.EXPECT().HasBalance(gomock.Any()).AnyTimes().Return(false)

	tx := &txStub{pubkey: []byte("some pubkey")}
	_, resp := app.OnCheckTx(context.Background(), tmtypes.RequestCheckTx{}, tx)
	assert.Equal(t, resp.Code, uint32(51))
}

func (s *AbciTestSuite) testOnCheckTxSuccessWithBalance(t *testing.T, app *processor.App, proc *procTest) {
	proc.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(false)
	proc.bank.EXPECT().HasBalance(gomock.Any()).AnyTimes().Return(true)

	tx := &txStub{pubkey: []byte("some pubkey")}
	_, resp := app.OnCheckTx(context.Background(), tmtypes.RequestCheckTx{}, tx)
	assert.Equal(t, resp.Code, uint32(0))
}

func (s *AbciTestSuite) testCheckSubmitOracleDataSucceedsWithValidOracleData(t *testing.T, app *processor.App, proc *procTest) {
	// given
	data, _ := json.Marshal(map[string]string{"BTC": "42"})
	tx := &txStub{
		cmd:  txn.SubmitOrderCommand,
		data: data,
	}

	// setup
	proc.oracles.Adaptors.EXPECT().Normalise(gomock.Any()).Return(&oracles.OracleData{}, nil)

	// when
	err := app.CheckSubmitOracleData(context.Background(), tx)

	// then
	assert.NoError(t, err)
}

func (s *AbciTestSuite) testCheckSubmitOracleDataForwardErrorWithInvalidOracleData(t *testing.T, app *processor.App, proc *procTest) {
	// given
	data, _ := json.Marshal(map[string]string{"BTC": "42"})
	tx := &txStub{
		cmd:  txn.SubmitOrderCommand,
		data: data,
	}

	// setup
	errInvalidOracleData := errors.New("invalid oracle data")
	proc.oracles.Adaptors.EXPECT().Normalise(gomock.Any()).Return(nil, errInvalidOracleData)

	// when
	err := app.CheckSubmitOracleData(context.Background(), tx)

	// then
	require.Error(t, err)
	assert.Equal(t, errInvalidOracleData, err)
}

func (s *AbciTestSuite) testDeliverSubmitOracleDataBroadcastValidOracleData(t *testing.T, app *processor.App, proc *procTest) {
	// given
	data, _ := json.Marshal(map[string]string{"BTC": "42"})
	tx := &txStub{
		cmd:  txn.SubmitOrderCommand,
		data: data,
	}

	// setup
	proc.oracles.Adaptors.EXPECT().Normalise(gomock.Any()).Return(&oracles.OracleData{}, nil)
	proc.oracles.Engine.EXPECT().BroadcastData(gomock.Any(), gomock.Any()).Times(1)

	// when
	err := app.DeliverSubmitOracleData(context.Background(), tx)

	// then
	assert.NoError(t, err)
}

func (s *AbciTestSuite) testDeliverSubmitOracleDataDoesNotBroadcastInvalidOracleData(t *testing.T, app *processor.App, proc *procTest) {
	// given
	data, _ := json.Marshal(map[string]string{"BTC": "42"})
	tx := &txStub{
		cmd:  txn.SubmitOrderCommand,
		data: data,
	}

	// setup
	proc.oracles.Adaptors.EXPECT().Normalise(gomock.Any()).Return(nil, errors.New("invalid oracle data"))
	proc.oracles.Engine.EXPECT().BroadcastData(gomock.Any(), gomock.Any()).Times(0)

	// when
	err := app.DeliverSubmitOracleData(context.Background(), tx)

	// then
	assert.Error(t, err)
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
		{"Call Begin and Commit - success", s.testBeginCommitSuccess},
		{"Call Begin twice, only calls commander once", s.testBeginCallsCommanderOnce},
		{"OnCheckTx fail with no balance", s.testOnCheckTxFailWithNoBalances},
		{"OnCheckTx success with balance", s.testOnCheckTxSuccessWithBalance},
		{
			"DeliverSubmitOracleData broadcasts valid oracle data through oracle engine",
			s.testDeliverSubmitOracleDataBroadcastValidOracleData,
		}, {
			"DeliverSubmitOracleData does not broadcast invalid oracle data through oracle engine",
			s.testDeliverSubmitOracleDataDoesNotBroadcastInvalidOracleData,
		},
		{
			"CheckSubmitOracleData succeeds with valid oracle data",
			s.testCheckSubmitOracleDataSucceedsWithValidOracleData,
		}, {
			"CheckSubmitOracleData forward error with invalid oracle data",
			s.testCheckSubmitOracleDataForwardErrorWithInvalidOracleData,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			proc := getTestProcessor(t)
			defer proc.ctrl.Finish()

			app := s.newApp(proc)
			test.fn(t, app, proc)
		})
	}
}

type txStub struct {
	pubkey []byte
	data   []byte
	cmd    txn.Command
}

func (tx *txStub) Command() txn.Command          { return tx.cmd }
func (tx *txStub) Unmarshal(v interface{}) error { return json.Unmarshal(tx.data, v) }
func (tx *txStub) PubKey() []byte                { return tx.pubkey }
func (tx *txStub) Party() string                 { return hex.EncodeToString(tx.pubkey) }
func (txStub) Hash() []byte                      { return nil }
func (txStub) Signature() []byte                 { return nil }
func (txStub) Validate() error                   { return nil }
func (txStub) BlockHeight() uint64               { return 0 }
