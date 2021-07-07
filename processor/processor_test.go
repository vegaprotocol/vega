package processor_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/processor/mocks"

	"github.com/golang/mock/gomock"
)

type procTest struct {
	eng     *mocks.MockExecutionEngine
	ts      *mocks.MockTimeService
	stat    *mocks.MockStats
	tickCB  func(context.Context, time.Time)
	ctrl    *gomock.Controller
	cmd     *mocks.MockCommander
	assets  *mocks.MockAssets
	top     *mocks.MockValidatorTopology
	gov     *mocks.MockGovernanceEngine
	notary  *mocks.MockNotary
	evtfwd  *mocks.MockEvtForwarder
	witness *mocks.MockWitness
	bank    *mocks.MockBanking
	netp    *mocks.MockNetworkParameters
	oracles *stubOracles
}

type stubWallet struct {
	key    []byte
	chain  string
	signed []byte
	err    error
}

type stubOracles struct {
	Engine   *mocks.MockOraclesEngine
	Adaptors *mocks.MockOracleAdaptors
}

func getTestProcessor(t *testing.T) *procTest {
	ctrl := gomock.NewController(t)
	eng := mocks.NewMockExecutionEngine(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	stat := mocks.NewMockStats(ctrl)
	cmd := mocks.NewMockCommander(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	top := mocks.NewMockValidatorTopology(ctrl)
	gov := mocks.NewMockGovernanceEngine(ctrl)
	notary := mocks.NewMockNotary(ctrl)
	evtfwd := mocks.NewMockEvtForwarder(ctrl)
	witness := mocks.NewMockWitness(ctrl)
	bank := mocks.NewMockBanking(ctrl)
	netp := mocks.NewMockNetworkParameters(ctrl)
	oracles := &stubOracles{
		Engine:   mocks.NewMockOraclesEngine(ctrl),
		Adaptors: mocks.NewMockOracleAdaptors(ctrl),
	}

	var cb func(context.Context, time.Time)
	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1).Do(func(c func(context.Context, time.Time)) {
		cb = c
	})

	return &procTest{
		eng:     eng,
		ts:      ts,
		stat:    stat,
		tickCB:  cb,
		ctrl:    ctrl,
		cmd:     cmd,
		assets:  assets,
		top:     top,
		gov:     gov,
		notary:  notary,
		evtfwd:  evtfwd,
		witness: witness,
		bank:    bank,
		netp:    netp,
		oracles: oracles,
	}
}
