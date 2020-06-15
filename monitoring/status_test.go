// +build !race

package monitoring_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/monitoring/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/p2p"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func TestAppStatus(t *testing.T) {

	t.Skip("This test needs to be written with careful regard to race conditions as exposed when using timers (#207, #282, #317)")

	log := logging.NewTestLogger()
	defer log.Sync()

	statusRes := tmctypes.ResultStatus{
		SyncInfo: tmctypes.SyncInfo{
			CatchingUp: false,
		},
		NodeInfo: p2p.DefaultNodeInfo{
			Version: "0.33.5",
		},
	}

	cfg := monitoring.NewDefaultConfig()
	cfg.Interval = encoding.Duration{Duration: 50 * time.Millisecond}

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

		checker := monitoring.New(log, cfg, blockchainClient)

		wg.Wait()
		assert.Equal(t, types.ChainStatus_CHAIN_STATUS_CONNECTED, checker.ChainStatus())

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
		blockchainClient.EXPECT().Health().Return(nil, nil)
		blockchainClient.EXPECT().GetStatus(gomock.Any()).Return(&statusRes2, nil).Do(func(_ context.Context) {
			// add the defer so that the waitgroup is only marked as done after we've returned the caching-up status
			defer wg.Done()
		})

		checker := monitoring.New(log, cfg, blockchainClient)

		wg.Wait()
		assert.Equal(t, types.ChainStatus_CHAIN_STATUS_REPLAYING, checker.ChainStatus())

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
		checker := monitoring.New(log, cfg, blockchainClient)
		<-end
		assert.Equal(t, types.ChainStatus_CHAIN_STATUS_DISCONNECTED, checker.ChainStatus())

		checker.Stop()
	})

	t.Run("Status = DISCONNECTED if Status previously = CONNECTED and client !healthy", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		blockchainClient := mocks.NewMockBlockchainClient(mockCtrl)

		wg := &sync.WaitGroup{}
		wg.Add(1)

		end := make(chan struct{})
		returnError := 0
		blockchainClient.EXPECT().Health().MinTimes(1).DoAndReturn(func() (*tmctypes.ResultHealth, error) {
			if returnError != 0 {
				defer func() { end <- struct{}{} }()
				return nil, errors.New("err")
			}
			returnError = 1
			return nil, nil
		})
		blockchainClient.EXPECT().GetStatus(gomock.Any()).Return(&statusRes, nil).Do(func(ctx context.Context) {
			wg.Done()
		})
		checker := monitoring.New(log, cfg, blockchainClient)
		wg.Wait()
		// ensure status is CONNECTED
		assert.Equal(t, types.ChainStatus_CHAIN_STATUS_CONNECTED, checker.ChainStatus())
		<-end
		// ensure it's now disconnected
		assert.Equal(t, types.ChainStatus_CHAIN_STATUS_DISCONNECTED, checker.ChainStatus())

		checker.Stop()

	})

}
