package processor_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/processor/mocks"

	"github.com/golang/mock/gomock"
)

type procTest struct {
	eng    *mocks.MockExecutionEngine
	ts     *mocks.MockTimeService
	stat   *mocks.MockStats
	tickCB func(context.Context, time.Time)
	ctrl   *gomock.Controller
	cmd    *mocks.MockCommander
	wallet *mocks.MockWallet
	assets *mocks.MockAssets
	top    *mocks.MockValidatorTopology
	gov    *mocks.MockGovernanceEngine
	notary *mocks.MockNotary
	evtfwd *mocks.MockEvtForwarder
	erc    *mocks.MockExtResChecker
	bank   *mocks.MockBanking
	netp   *mocks.MockNetworkParameters
}

type stubWallet struct {
	key    []byte
	chain  string
	signed []byte
	err    error
}

func getTestProcessor(t *testing.T) *procTest {
	ctrl := gomock.NewController(t)
	eng := mocks.NewMockExecutionEngine(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	stat := mocks.NewMockStats(ctrl)
	cmd := mocks.NewMockCommander(ctrl)
	wallet := mocks.NewMockWallet(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	top := mocks.NewMockValidatorTopology(ctrl)
	gov := mocks.NewMockGovernanceEngine(ctrl)
	notary := mocks.NewMockNotary(ctrl)
	evtfwd := mocks.NewMockEvtForwarder(ctrl)
	erc := mocks.NewMockExtResChecker(ctrl)
	bank := mocks.NewMockBanking(ctrl)
	netp := mocks.NewMockNetworkParameters(ctrl)

	//top.EXPECT().Ready().AnyTimes().Return(true)
	var cb func(context.Context, time.Time)
	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1).Do(func(c func(context.Context, time.Time)) {
		cb = c
	})
	wal := getTestStubWallet()
	wallet.EXPECT().Get(nodewallet.Vega).AnyTimes().Return(wal, true)
	top.EXPECT().IsValidator().AnyTimes().Return(true)

	return &procTest{
		eng:    eng,
		ts:     ts,
		stat:   stat,
		tickCB: cb,
		ctrl:   ctrl,
		cmd:    cmd,
		wallet: wallet,
		assets: assets,
		top:    top,
		gov:    gov,
		notary: notary,
		evtfwd: evtfwd,
		erc:    erc,
		bank:   bank,
		netp:   netp,
	}
}

func getTestStubWallet() *stubWallet {
	return &stubWallet{
		key:   []byte("test key"),
		chain: string(nodewallet.Vega),
	}
}

func (s stubWallet) Chain() string {
	return s.chain
}

func (s stubWallet) Algo() string {
	return "vega/ed25519"
}

func (s stubWallet) Version() uint64 {
	return 1
}

func (s stubWallet) PubKeyOrAddress() []byte {
	return s.key
}

func (s stubWallet) Sign(_ []byte) ([]byte, error) {
	return s.signed, s.err
}
