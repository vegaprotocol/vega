package monitoring

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
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
		checker := NewStatusChecker(log, blockchainClient, 50*time.Nanosecond)
		wg.Wait()
		assert.Equal(t, types.ChainStatus_CONNECTED, checker.Blockchain.Status())

		checker.Blockchain.Stop()
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

		checker := NewStatusChecker(log, blockchainClient, 10*time.Millisecond)
		wg.Wait()
		assert.Equal(t, types.ChainStatus_REPLAYING, checker.Blockchain.Status())

		checker.Blockchain.Stop()
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

		checker := NewStatusChecker(log, blockchainClient, 50*time.Nanosecond)
		<-end
		assert.Equal(t, types.ChainStatus_DISCONNECTED, checker.Blockchain.Status())

		checker.Blockchain.Stop()
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

		checker := NewStatusChecker(log, blockchainClient, 10*time.Millisecond)
		wg.Wait()
		// ensure status is CONNECTED
		assert.Equal(t, types.ChainStatus_CONNECTED, checker.Blockchain.Status())
		atomic.StoreUint32(&returnError, 1)
		<-end
		// ensure it's now disconnected
		assert.Equal(t, types.ChainStatus_DISCONNECTED, checker.Blockchain.Status())

		checker.Blockchain.Stop()
		checker.Stop()

	})

}
