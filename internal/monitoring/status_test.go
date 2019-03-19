package monitoring

import (
	"errors"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"

	"code.vegaprotocol.io/vega/internal/blockchain/mocks"
	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		chainClient := &mocks.Client{}
		chainClient.On("Health").Return(nil, nil)
		chainClient.On("GetStatus", mock.AnythingOfType("*context.emptyCtx")).Return(&statusRes, nil)

		checker := NewStatusChecker(log, chainClient, 1*time.Nanosecond)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, types.ChainStatus_CONNECTED, checker.Blockchain.Status())

		checker.Blockchain.Stop()
		checker.Stop()
	})

	t.Run("Status = REPLAY if client healthy + catching up", func(t *testing.T) {
		chainClient := &mocks.Client{}
		statusRes2 := statusRes
		statusRes2.SyncInfo.CatchingUp = true

		chainClient.On("Health").Return(nil, nil)
		chainClient.On("GetStatus", mock.AnythingOfType("*context.emptyCtx")).Return(&statusRes2, nil)

		checker := NewStatusChecker(log, chainClient, 1*time.Nanosecond)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, types.ChainStatus_REPLAYING, checker.Blockchain.Status())

		checker.Blockchain.Stop()
		checker.Stop()
	})

	t.Run("Status = DISCONNECTED if client !healthy", func(t *testing.T) {
		chainClient := &mocks.Client{}
		chainClient.On("Health").Return(nil, errors.New("err"))

		checker := NewStatusChecker(log, chainClient, 1*time.Nanosecond)

		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, types.ChainStatus_DISCONNECTED, checker.Blockchain.Status())

		checker.Blockchain.Stop()
		checker.Stop()
	})

	t.Run("Status = DISCONNECTED if Status previously = DISCONNECTED and client !healthy", func(t *testing.T) {
		chainClient := &mocks.Client{}

		chainClient.On("Health").Return(nil, nil)
		chainClient.On("GetStatus",
			mock.AnythingOfType("*context.emptyCtx")).Return(&statusRes, nil)

		checker := NewStatusChecker(log, chainClient, 1*time.Nanosecond)

		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, types.ChainStatus_CONNECTED, checker.Blockchain.Status())

		chainClient = &mocks.Client{}
		chainClient.On("Health").Return(nil,
			errors.New("phobos connection link lost"))
		chainClient.On("GetStatus",
			mock.AnythingOfType("*context.emptyCtx")).Return(&statusRes, nil)

		checker.Blockchain.SetClient(chainClient)

		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, types.ChainStatus_DISCONNECTED, checker.Blockchain.Status())

		checker.Blockchain.Stop()
		checker.Stop()
	})
}
