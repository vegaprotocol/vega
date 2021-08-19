package epochs_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/markets"
	"code.vegaprotocol.io/data-node/markets/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	*markets.Svc
	ctx         context.Context
	cfunc       context.CancelFunc
	log         *logging.Logger
	ctrl        *gomock.Controller
	order       *mocks.MockOrderStore
	market      *mocks.MockMarketStore
	marketDepth *mocks.MockMarketDepth
	marketData  *mocks.MockMarketDataStore
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	order := mocks.NewMockOrderStore(ctrl)
	market := mocks.NewMockMarketStore(ctrl)
	marketdata := mocks.NewMockMarketDataStore(ctrl)
	marketdepth := mocks.NewMockMarketDepth(ctrl)
	log := logging.NewTestLogger()
	ctx, cfunc := context.WithCancel(context.Background())
	svc, err := markets.NewService(
		log,
		markets.NewDefaultConfig(),
		market,
		order,
		marketdata,
		marketdepth,
	)
	assert.NoError(t, err)
	return &testService{
		Svc:         svc,
		ctx:         ctx,
		cfunc:       cfunc,
		log:         log,
		ctrl:        ctrl,
		order:       order,
		market:      market,
		marketDepth: marketdepth,
		marketData:  marketdata,
	}
}

func TestEpochService_GetAll(t *testing.T) {
	// @TODO
}

func (t *testService) Finish() {
	t.cfunc()
	_ = t.log.Sync()
	t.ctrl.Finish()
}
