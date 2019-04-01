package monitoring_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/monitoring/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func TestAppStatus(t *testing.T) {
	log := logging.NewLoggerFromEnv("dev")
	defer log.Sync()

	statusRes := tmctypes.ResultStatus{
		SyncInfo: tmctypes.SyncInfo{
			CatchingUp: false,
		},
	}

	cfg := &Config{
		log:                  log,
		IntervalMilliseconds: 50 * time.Millisecond,
		Retries:              5,
	}

	t.Run("Status = CONNECTED if client healthy + !catching up", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		blockchainClient := mocks.NewMockBlockchainClient(mockCtrl)

		wg := &sync.WaitGroup{}
		wg.Add(1)
		blockchainClient.EXPECT().Health().MinTimes(1).Return(nil, nil)
		blockchainClient.EXPECT().GetStatus(gomock.Any()).Return(&statusRes, nil).Do(func(ctx context.Context) {
			wg.Done()
		})

		checker := monitoring.New(cfg, blockchainClient)

		wg.Wait()
		assert.Equal(t, types.ChainStatus_CONNECTED, checker.ChainStatus())

		checker.Stop()
	})

	t.Run("Status = REPLAY if client healthy + catching up", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		blockchainClient := mocks.NewMockBlockchainClient(mockCtrl)

		statusRes2 := statusRes
		statusRes2.SyncInfo.CatchingUp = true

		wg := &sync.WaitGroup{}
		wg.Add(1)
		blockchainClient.EXPECT().Health().MinTimes(1).Return(nil, nil)
		blockchainClient.EXPECT().GetStatus(gomock.Any()).Return(&statusRes2, nil).Do(func(ctx context.Context) {
			wg.Done()
		})

		checker := monitoring.New(cfg, blockchainClient)

		wg.Wait()
		assert.Equal(t, types.ChainStatus_REPLAYING, checker.ChainStatus())

		checker.Stop()
	})

	t.Run("Status = DISCONNECTED if client !healthy", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		blockchainClient := mocks.NewMockBlockchainClient(mockCtrl)

		end := make(chan struct{})
		blockchainClient.EXPECT().Health().MinTimes(1).Return(nil, errors.New("err")).Do(func() {
			end <- struct{}{}
		})
		checker := monitoring.New(cfg, blockchainClient)
		<-end
		assert.Equal(t, types.ChainStatus_DISCONNECTED, checker.ChainStatus())

		checker.Stop()
	})

	t.Run("Status = DISCONNECTED if Status previously = CONNECTED and client !healthy", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		blockchainClient := mocks.NewMockBlockchainClient(mockCtrl)

		wg := &sync.WaitGroup{}
		wg.Add(1)

		end := make(chan struct{})
		var returnError uint32
		blockchainClient.EXPECT().Health().MinTimes(1).DoAndReturn(func() (*tmctypes.ResultHealth, error) {
			if atomic.LoadUint32(&returnError) != 0 {
				defer func() { end <- struct{}{} }()
				return nil, errors.New("err")
			}
			return nil, nil
		})
		blockchainClient.EXPECT().GetStatus(gomock.Any()).Return(&statusRes, nil).Do(func(ctx context.Context) {
			wg.Done()
		})
		checker := monitoring.New(cfg, blockchainClient)
		wg.Wait()
		// ensure status is CONNECTED
		assert.Equal(t, types.ChainStatus_CONNECTED, checker.ChainStatus())
		atomic.StoreUint32(&returnError, 1)
		<-end
		// ensure it's now disconnected
		assert.Equal(t, types.ChainStatus_DISCONNECTED, checker.ChainStatus())

		checker.Stop()

	})

}
