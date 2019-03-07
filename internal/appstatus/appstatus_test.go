package appstatus

import (
	"errors"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/blockchain/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	t.Run("Status = CONNECTED if client healthy + !catchingup", func(t *testing.T) {
		chainclt := &mocks.Client{}
		tickerDuration = 1 * time.Nanosecond
		chainclt.On("Health").Return(nil, nil)
		chainclt.On("GetStatus", mock.AnythingOfType("*context.emptyCtx")).Return(&statusRes, nil)

		appst := New(log, chainclt)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, proto.AppStatus_CONNECTED, appst.Get())

		appst.Stop()
	})

	t.Run("Status = REPLAY if client healthy + catchingup", func(t *testing.T) {
		chainclt := &mocks.Client{}
		tickerDuration = 1 * time.Nanosecond
		statusRes2 := statusRes
		statusRes2.SyncInfo.CatchingUp = true

		chainclt.On("Health").Return(nil, nil)
		chainclt.On("GetStatus", mock.AnythingOfType("*context.emptyCtx")).Return(&statusRes2, nil)
		appst := New(log, chainclt)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, proto.AppStatus_CHAIN_REPLAYING, appst.Get())

		appst.Stop()
	})

	t.Run("Status = DISCONNECTED if client !healthy", func(t *testing.T) {
		chainclt := &mocks.Client{}
		tickerDuration = 1 * time.Nanosecond
		chainclt.On("Health").Return(nil, errors.New("err"))

		appst := New(log, chainclt)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, proto.AppStatus_DISCONNECTED, appst.Get())

		appst.Stop()
	})

}
