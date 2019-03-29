package blockchain

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/blockchain/mocks"
	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testApp struct {
	*AbciApplication
	errCh chan struct{}
	ctrl  *gomock.Controller
	log   *logging.Logger
	proc  *mocks.MockApplicationProcessor
	time  *mocks.MockApplicationTime
	svc   *mocks.MockApplicationService
}

func getTestApp(t *testing.T) *testApp {
	ctrl := gomock.NewController(t)
	proc := mocks.NewMockApplicationProcessor(ctrl)
	svc := mocks.NewMockApplicationService(ctrl)
	time := mocks.NewMockApplicationTime(ctrl)
	log := logging.NewLoggerFromEnv("env")
	ch := make(chan struct{}, 1)
	errCb := func() {
		ch <- struct{}{}
	}
	app := NewAbciApplication(
		NewDefaultConfig(log),
		NewStats(),
		proc,
		svc,
		time,
		errCb,
	)
	return &testApp{
		AbciApplication: app,
		errCh:           ch,
		ctrl:            ctrl,
		log:             log,
		proc:            proc,
		time:            time,
		svc:             svc,
	}
}

func TestNewAbciApplication(t *testing.T) {
	app := getTestApp(t)
	defer app.Finish()
	assert.Equal(t, uint64(0), app.AbciApplication.stats.height)
}

func (t *testApp) Finish() {
	t.log.Sync()
	t.ctrl.Finish()
	close(t.errCh)
}
