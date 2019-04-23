package blockchain_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/blockchain/mocks"
	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testApp struct {
	*blockchain.AbciApplication
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
	log := logging.NewTestLogger()
	ch := make(chan struct{}, 1)
	errCb := func() {
		ch <- struct{}{}
	}
	app := blockchain.NewApplication(
		log,
		blockchain.NewDefaultConfig(),
		blockchain.NewStats(),
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

func TestNewApplication(t *testing.T) {
	app := getTestApp(t)
	defer app.Finish()
	assert.NotNil(t, app.Stats())
}

func (t *testApp) Finish() {
	t.log.Sync()
	t.ctrl.Finish()
	close(t.errCh)
}
