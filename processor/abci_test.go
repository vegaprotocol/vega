package processor_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/processor"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	tmtypes "github.com/tendermint/tendermint/abci/types"
	tmprototypes "github.com/tendermint/tendermint/proto/types"
)

type AbciTestSuite struct {
	app *processor.App
}

func (s *AbciTestSuite) newApp(proc *procTest) error {
	var err error
	s.app, err = processor.NewApp(
		logging.NewTestLogger(),
		processor.NewDefaultConfig(),
		nil,
		proc.assets,
		proc.bank,
		proc.erc,
		proc.evtfwd,
		proc.eng,
		proc.cmd,
		proc.col,
		proc.gov,
		proc.notary,
		proc.stat,
		proc.ts,
		proc.top,
		proc.wallet,
	)
	return err
}

func (s *AbciTestSuite) testCommitSuccess(t *testing.T, proc *procTest) {
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
	s.app.OnBeginBlock(tmtypes.RequestBeginBlock{
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

	s.app.OnCommit()
}

func TestAbci(t *testing.T) {
	s := &AbciTestSuite{}

	t.Run("OnBeginSuccess", func(t *testing.T) {
		proc := getTestProcessor(t)
		defer proc.ctrl.Finish()
		s.newApp(proc)

		s.testCommitSuccess(t, proc)
	})
}
