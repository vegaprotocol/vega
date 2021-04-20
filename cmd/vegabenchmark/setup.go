package main

import (
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/mock/gomock"
)

func setupVega() (*processor.App, error) {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())

	ctrl := gomock.NewController(&nopeTestReporter{log})
	nodeWallet := mocks.NewNodeWalletMock(ctrl)
	broker := mocks.NewBrokerMock(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	collateral := collateral.New(
		log,
		collateral.NewDefaultConfig(),
		broker,
		time.Time{},
	)
	_ = collateral
	timeService := vegatime.New(vegatime.NewDefaultConfig())
	// banking := banking.New(
	// 	log,
	// 	banking.NewDefaultConfig(), col banking.Collateral, erc banking.ExtResChecker, tsvc banking.TimeService, assets banking.Assets, notary banking.Notary, broker banking.Broker)

	assets := assets.New(
		log,
		assets.NewDefaultConfig(),
		nodeWallet,
		timeService)
	_ = assets

	// app, err := processor.NewApp(
	// 	log,
	// 	processor.NewDefaultConfig(),
	// 	func() { panic("cancel called") },
	// 	assets,
	// 	banking,
	// )

	return nil, nil
}

type nopeTestReporter struct{ log *logging.Logger }

func (n *nopeTestReporter) Errorf(format string, args ...interface{}) {
	n.log.Errorf(format, args...)
}
func (n *nopeTestReporter) Fatalf(format string, args ...interface{}) {
	n.log.Errorf(format, args...)
}
