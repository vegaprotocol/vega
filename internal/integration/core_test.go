package core_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/internal/collateral"
	// cmocks "code.vegaprotocol.io/vega/internal/collateral/mocks"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/execution/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/matching"
	"code.vegaprotocol.io/vega/internal/positions"
	"code.vegaprotocol.io/vega/internal/risk"
	"code.vegaprotocol.io/vega/internal/settlement"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/proto"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/gherkin"
	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
)

type traderState struct {
	pos             int
	margin, general int64
	markPrice       int
	gAcc            *proto.Account
	mAcc            *proto.Account
}

type tstReporter struct {
	err  error
	step string
}

type tstSetup struct {
	market   *proto.Market
	ctrl     *gomock.Controller
	core     *execution.Market
	candles  *mocks.MockCandleStore
	orders   *orderStub
	trades   *tradeStub
	parties  *mocks.MockPartyStore
	transfer *mocks.MockTransferResponseStore
	accounts *accStub
	// accounts   *cmocks.MockAccountBuffer
	accountIDs map[string]struct{}
	traderAccs map[string]map[proto.AccountType]*proto.Account

	// we need to call this engine directly
	colE *collateral.Engine
}

var (
	setup    *tstSetup
	core     *execution.Market
	accounts *storage.Account
	reporter tstReporter
)

func getMock(market *proto.Market) *tstSetup {
	if setup != nil {
		setup.ctrl.Finish()
		setup = nil // ready for GC
	}
	// the controller needs the reporter to report on errors or clunk out with fatal
	ctrl := gomock.NewController(&reporter)
	candles := mocks.NewMockCandleStore(ctrl)
	// orders := mocks.NewMockOrderStore(ctrl)
	orders := NewOrderStub()
	// trades := mocks.NewMockTradeStore(ctrl)
	trades := NewTradeStub()
	parties := mocks.NewMockPartyStore(ctrl)
	accounts := NewAccountStub()
	// accounts := cmocks.NewMockAccountBuffer(ctrl)
	transfer := mocks.NewMockTransferResponseStore(ctrl)
	colE, _ := collateral.New(
		logging.NewTestLogger(),
		collateral.NewDefaultConfig(),
		accounts,
		time.Now(),
	)
	// mock call to get the last candle
	candles.EXPECT().FetchLastCandle(gomock.Any(), gomock.Any()).MinTimes(1).Return(&proto.Candle{}, nil)

	setup := &tstSetup{
		market:     market,
		ctrl:       ctrl,
		candles:    candles,
		orders:     orders,
		trades:     trades,
		parties:    parties,
		transfer:   transfer,
		accounts:   accounts,
		accountIDs: map[string]struct{}{},
		traderAccs: map[string]map[proto.AccountType]*proto.Account{},
		colE:       colE,
	}

	return setup
}

func theMarket(mSetup *gherkin.DataTable) error {
	var (
		market                   string
		markPrice                uint64 // @TODO this needs to be part of the market definition somehow
		forward                  proto.Forward
		release, initial, search float64
	)
	forward.Params = &proto.ModelParamsBS{}
	for _, row := range mSetup.Rows {
		// skip header
		if row.Cells[0].Value == "name" {
			continue
		}
		// | name      | markprice | risk model | lamd | tau         | mu | r | sigma     | release factor | initial factor | search factor |
		market = row.Cells[0].Value
		markPrice, _ = strconv.ParseUint(row.Cells[1].Value, 10, 64)
		forward.Lambd, _ = strconv.ParseFloat(row.Cells[3].Value, 64)
		forward.Tau, _ = strconv.ParseFloat(row.Cells[4].Value, 64)
		forward.Params.Mu, _ = strconv.ParseFloat(row.Cells[5].Value, 64)
		forward.Params.R, _ = strconv.ParseFloat(row.Cells[6].Value, 64)
		forward.Params.Sigma, _ = strconv.ParseFloat(row.Cells[7].Value, 64)
		release, _ = strconv.ParseFloat(row.Cells[8].Value, 64)
		initial, _ = strconv.ParseFloat(row.Cells[9].Value, 64)
		search, _ = strconv.ParseFloat(row.Cells[10].Value, 64)
	}
	parts := strings.Split(market, "/")
	mkt := &proto.Market{
		Id:   market,
		Name: market,
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:               fmt.Sprintf("Crypto/%s/Futures/%s", parts[0], parts[1]),
				Code:             fmt.Sprintf("FX:%s%s", parts[0], parts[1]),
				Name:             "December 2019 test future",
				InitialMarkPrice: markPrice,
				Metadata: &proto.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &proto.Instrument_Future{
					Future: &proto.Future{
						Maturity: "2019-12-31T00:00:00Z",
						Oracle: &proto.Future_EthereumEvent{
							EthereumEvent: &proto.EthereumEvent{
								ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
								Event:      "price_changed",
							},
						},
						Asset: parts[0],
					},
				},
			},
			RiskModel: &proto.TradableInstrument_Forward{
				Forward: &forward,
			},
			MarginCalculator: &proto.MarginCalculator{
				ScalingFactors: &proto.ScalingFactors{
					SearchLevel:       search,
					InitialMargin:     initial,
					CollateralRelease: release,
				},
			},
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}
	//   mu=0,r=0,sigma=3.6907199
	// tau: 0.000114077
	// lamd: .01
	log := logging.NewTestLogger()
	// the controller needs the reporter to report on errors or clunk out with fatal
	setup = getMock(mkt)
	party := execution.NewParty(log, setup.colE, []proto.Market{*mkt}, setup.parties)
	m, err := execution.NewMarket(
		log,
		risk.NewDefaultConfig(),
		positions.NewDefaultConfig(),
		settlement.NewDefaultConfig(),
		matching.NewDefaultConfig(),
		setup.colE,
		party, // party-engine here!
		mkt,
		setup.candles,
		setup.orders,
		setup.parties,
		setup.trades,
		setup.transfer,
		time.Now(),
	)
	if err != nil {
		return err
	}
	setup.core = m
	core = m
	return nil
}

func theSystemAccounts(systemAccounts *gherkin.DataTable) error {
	// we currently have N accounts, creating system accounts should create 2 more accounts
	current := len(setup.accounts.data)
	// this should create market accounts, currently same way it's done in execution engine (register market)
	asset, _ := setup.market.GetAsset()
	_, _ = setup.colE.CreateMarketAccounts(setup.core.GetID(), asset, 0)
	if len(setup.accounts.data) != current+2 {
		reporter.err = fmt.Errorf("error creating system accounts")
	}
	return reporter.err
}

func tradersHaveTheFollowingState(traders *gherkin.DataTable) error {
	// this is going to be tricky... we have no product set up, we can only ensure the trader accounts are created,  but that's about it...
	// damn... positions engine is not open here, let's just ram through the trades, and update the balances after the fact
	market := core.GetID()
	maxPos := 100 // ensure we can move 100 positions either long or short, doesn't really matter which way
	// traderStates := map[string]traderState{}
	// tomorrow := time.Now().Add(time.Hour * 24)
	// each position will be put down as an order
	// orders := make([]*proto.Order, 0, len(traders.Rows))
	for _, row := range traders.Rows {
		// skip first row
		if row.Cells[0].Value == "trader" {
			continue
		}
		// it's safe to ignore this error for now
		pos, err := strconv.Atoi(row.Cells[1].Value)
		if err != nil {
			return err
		}
		// mark, err := strconv.Atoi(row.Cells[5].Value)
		// if err != nil {
		// return err
		// }
		marginBal, err := strconv.ParseInt(row.Cells[2].Value, 10, 64)
		if err != nil {
			return err
		}
		generalBal, err := strconv.ParseInt(row.Cells[3].Value, 10, 64)
		if err != nil {
			return err
		}
		// highest net pos
		if pos > maxPos {
			maxPos = pos
		}
		asset, _ := setup.market.GetAsset()
		margin, general := setup.colE.CreateTraderAccount(row.Cells[0].Value, market, asset)
		_ = setup.colE.IncrementBalance(margin, marginBal)
		_ = setup.colE.IncrementBalance(general, generalBal)
		// add trader accounts to map - this is the state they should have now
		setup.traderAccs[row.Cells[0].Value] = map[proto.AccountType]*proto.Account{
			proto.AccountType_MARGIN: &proto.Account{
				Id:      margin,
				Type:    proto.AccountType_MARGIN,
				Balance: marginBal,
			},
			proto.AccountType_GENERAL: &proto.Account{
				Id:      general,
				Type:    proto.AccountType_GENERAL,
				Balance: generalBal,
			},
		}
		// this is creating the positions and setting mark price... we can't do that just yet here
		// make sure there's ample margin balance to get to the positions we need
		// 	// add to states
		// 	side := proto.Side_Buy
		// 	vol := ts.pos
		// 	if pos < 0 {
		// 		side = proto.Side_Sell
		// 		// absolute value for volume
		// 		vol *= -1
		// 	}
		// 	traderStates[row.Cells[0].Value] = ts
		// 	order := proto.Order{
		// 		Id:        uuid.NewV4().String(),
		// 		MarketID:  market,
		// 		PartyID:   row.Cells[0].Value,
		// 		Side:      side,
		// 		Price:     1,
		// 		Size:      uint64(vol),
		// 		ExpiresAt: tomorrow.Unix(),
		// 	}
		// get order ready to submit
		// orders = append(orders, &order)
	}
	// ok, submit some 'fake' orders, ensuring that the traders' positions all match up
	// for _, o := range orders {
	// 	if _, err := core.SubmitOrder(o); err != nil {
	// 		return err
	// 	}
	// }
	// update their account balances, so we have established the traders' states
	// for _, ts := range traderStates {
	// if err := accounts.UpdateBalance(ts.gAcc.Id, ts.general); err != nil {
	// return err
	// }
	// if err := accounts.UpdateBalance(ts.mAcc.Id, ts.margin); err != nil {
	// return err
	// }
	// }
	return nil
}

func theFollowingOrders(orderT *gherkin.DataTable) error {
	tomorrow := time.Now().Add(time.Hour * 24)
	core := setup.core
	market := core.GetID()
	calls := len(orderT.Rows)
	// if the first row is a header row, exclude from the call count
	if orderT.Rows[0].Cells[0].Value == "trader" {
		calls--
	}
	// build + place all orders
	for _, row := range orderT.Rows {
		if row.Cells[0].Value == "trader" {
			continue
		}
		// else expect call to get party
		setup.parties.EXPECT().GetByID(row.Cells[0].Value).Times(1).Return(
			&proto.Party{
				Id: row.Cells[0].Value,
			},
			nil,
		)

		side := proto.Side_Buy
		if row.Cells[1].Value == "sell" {
			side = proto.Side_Sell
		}
		vol, err := strconv.Atoi(row.Cells[2].Value)
		if err != nil {
			return err
		}
		price, err := strconv.ParseInt(row.Cells[3].Value, 10, 64)
		if err != nil {
			return err
		}
		expTrades, err := strconv.Atoi(row.Cells[4].Value)
		if err != nil {
			return err
		}
		order := proto.Order{
			Id:          uuid.NewV4().String(),
			MarketID:    market,
			PartyID:     row.Cells[0].Value,
			Side:        side,
			Price:       uint64(price),
			Size:        uint64(vol),
			Remaining:   uint64(vol),
			ExpiresAt:   tomorrow.Unix(),
			Type:        proto.Order_LIMIT,
			TimeInForce: proto.Order_GTT,
			CreatedAt:   time.Now().UnixNano(),
		}
		result, err := core.SubmitOrder(&order)
		if err != nil {
			return err
		}
		if len(result.Trades) != expTrades {
			return fmt.Errorf("expected %d trades, instead saw %d (%#v)", expTrades, len(result.Trades), *result)
		}
	}
	return nil
}

func tradersLiability(liablityTbl *gherkin.DataTable) error {
	for _, row := range liablityTbl.Rows {
		// skip header
		if row.Cells[0].Value == "trader" {
			continue
		}
		trader := row.Cells[0].Value
		margin, err := strconv.ParseInt(row.Cells[4].Value, 10, 64)
		if err != nil {
			return err
		}
		general, err := strconv.ParseInt(row.Cells[5].Value, 10, 64)
		if err != nil {
			return err
		}
		accounts := setup.traderAccs[trader]
		acc, err := setup.colE.GetAccountByID(accounts[proto.AccountType_MARGIN].Id)
		if err != nil {
			return err
		}
		// sync margin account state
		setup.traderAccs[trader][proto.AccountType_MARGIN] = acc
		if acc.Balance != margin {
			return fmt.Errorf("expected %s margin account balance to be %d instead saw %d", trader, margin, acc.Balance)
		}
		acc, err = setup.colE.GetAccountByID(accounts[proto.AccountType_GENERAL].Id)
		if err != nil {
			return err
		}
		if acc.Balance != general {
			return fmt.Errorf("expected %s general account balance to be %d, instead saw %d", trader, general, acc.Balance)
		}
		// sync general account state
		setup.traderAccs[trader][proto.AccountType_GENERAL] = acc
	}
	return nil
}

func hasNotBeenAddedToTheMarket(trader string) error {
	accounts := setup.traderAccs[trader]
	acc, err := setup.colE.GetAccountByID(accounts[proto.AccountType_MARGIN].Id)
	if err != nil || acc.Balance == 0 {
		return nil
	}
	return fmt.Errorf("didn't expect %s to hava a margin account with balance, instead saw %d", trader, acc.Balance)
}

func iCheckTheUpdatedBalancesAndPositions() error {
	// not sure if we need this step
	return nil
	// return godog.ErrPending
}

func iExpectToSee(arg1 *gherkin.DataTable) error {
	return godog.ErrPending
}

func FeatureContext(s *godog.Suite) {
	// each step changes the output from the reporter
	// so we know where a mock failed
	s.BeforeStep(func(step *gherkin.Step) {
		// rm any errors from previous step (if applies)
		reporter.err = nil
		reporter.step = step.Text
	})
	// if a mock assert failed, we're just setting an error here and crash out of the test here
	s.AfterStep(func(step *gherkin.Step, err error) {
		if err != nil && reporter.err == nil {
			reporter.err = err
		}
		if reporter.err != nil {
			reporter.Fatalf("some mock assertion failed: %v", reporter.err)
		}
	})
	s.Step(`^the market:$`, theMarket)
	s.Step(`^the system accounts:$`, theSystemAccounts)
	s.Step(`^traders have the following state:$`, tradersHaveTheFollowingState)
	s.Step(`^the following orders:$`, theFollowingOrders)
	s.Step(`^I place the following orders:$`, theFollowingOrders)
	s.Step(`^I expect the trader to have a margin liability:$`, tradersLiability)
	s.Step(`^"([^"]*)" has not been added to the market$`, hasNotBeenAddedToTheMarket)
	s.Step(`^I check the updated balances and positions$`, iCheckTheUpdatedBalancesAndPositions)
	s.Step(`^I expect to see:$`, iExpectToSee)
}

func (t tstReporter) Errorf(format string, args ...interface{}) {
	fmt.Printf("%s ERROR: %s", t.step, fmt.Sprintf(format, args...))
}

func (t tstReporter) Fatalf(format string, args ...interface{}) {
	fmt.Printf("%s FATAL: %s", t.step, fmt.Sprintf(format, args...))
	os.Exit(1)
}
