// +build ignore

package core_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/internal/collateral"
	cmocks "code.vegaprotocol.io/vega/internal/collateral/mocks"
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
	// _ "github.com/DATA-DOG/godog/cmd/godog"
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
	market     *proto.Market
	ctrl       *gomock.Controller
	core       *execution.Market
	candles    *mocks.MockCandleStore
	orders     *mocks.MockOrderStore
	trades     *mocks.MockTradeStore
	parties    *mocks.MockPartyStore
	accounts   *cmocks.MockAccountBuffer
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
	orders := mocks.NewMockOrderStore(ctrl)
	trades := mocks.NewMockTradeStore(ctrl)
	parties := mocks.NewMockPartyStore(ctrl)
	accounts := cmocks.NewMockAccountBuffer(ctrl)
	colE, _ := collateral.New(
		logging.NewTestLogger(),
		collateral.NewDefaultConfig(),
		accounts,
		time.Now(),
	)

	setup := &tstSetup{
		market:     market,
		ctrl:       ctrl,
		candles:    candles,
		orders:     orders,
		trades:     trades,
		parties:    parties,
		accounts:   accounts,
		accountIDs: map[string]struct{}{},
		traderAccs: map[string]map[proto.AccountType]*proto.Account{},
		colE:       colE,
	}

	return setup
}

func theMarket(market string) error {
	parts := strings.Split(market, "/")
	mkt := &proto.Market{
		Id:   market,
		Name: market,
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:   fmt.Sprintf("Crypto/%s/Futures/%s", parts[0], parts[1]),
				Code: fmt.Sprintf("FX:%s%s", parts[0], parts[1]),
				Name: "December 2019 test future",
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
						Asset: "ETH",
					},
				},
			},
			RiskModel: &proto.TradableInstrument_Forward{
				Forward: &proto.Forward{
					Lambd: 0.01,
					Tau:   1.0 / 365.25 / 24,
					Params: &proto.ModelParamsBS{
						Mu:    0,
						R:     0.016,
						Sigma: 0.09,
					},
				},
			},
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}
	log := logging.NewTestLogger()
	// the controller needs the reporter to report on errors or clunk out with fatal
	setup := getMock(mkt)
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
		time.Now(),
		1, // seq?
	)
	if err != nil {
		return err
	}
	setup.core = m
	core = m
	return nil
}

func theSystemAccounts(systemAccounts *gherkin.DataTable) error {
	// we're expecting 2 accounts to be created: a system insurance and general account
	// types required
	reqT := map[proto.AccountType]bool{
		proto.AccountType_SETTLEMENT: false,
		proto.AccountType_INSURANCE:  false,
	}
	setup.accounts.EXPECT().Add(gomock.Any()).Times(2).Do(func(a proto.Account) {
		setup.accountIDs[a.Id] = struct{}{}
		if _, ok := reqT[a.Type]; !ok {
			reporter.err = fmt.Errorf("account type %s unexpectedly created when creating system accounts", a.Type.String())
		} else {
			reqT[a.Type] = true
		}
	})
	// this should create market accounts, currently same way it's done in execution engine (register market)
	// we can ignore the lines here safely
	asset, _ := setup.market.GetAsset()
	_, _ = setup.colE.CreateMarketAccounts(setup.core.GetID(), asset, 0)
	for t, ok := range reqT {
		if !ok && reporter.err == nil {
			reporter.err = fmt.Errorf("creating system accounts failed to create %s account", t.String())
		}
	}
	return reporter.err
}

func tradersHaveTheFollowingState(traders *gherkin.DataTable) error {
	// this is going to be tricky... we have no product set up, we can only ensure the trader accounts are created,  but that's about it...
	// damn... positions engine is not open here, let's just ram through the trades, and update the balances after the fact
	market := core.GetID()
	maxPos := 100 // ensure we can move 100 positions either long or short, doesn't really matter which way
	traderStates := map[string]traderState{}
	tomorrow := time.Now().Add(time.Hour * 24)
	// each position will be put down as an order
	orders := make([]*proto.Order, 0, len(traders.Rows))
	for _, row := range traders.Rows {
		// skip first row
		if row.Cells[0].Value == "trader" {
			continue
		}
		// it's safe to ignore this error for now
		// pos, err := strconv.Atoi(row.Cells[1].Value)
		// if err != nil {
		// return err
		// }
		// mark, err := strconv.Atoi(row.Cells[5].Value)
		// if err != nil {
		// return err
		// }
		margin, err := strconv.ParseInt(row.Cells[2].Value, 10, 64)
		if err != nil {
			return err
		}
		general, err := strconv.ParseInt(row.Cells[3].Value, 10, 64)
		if err != nil {
			return err
		}
		// highest net pos
		if pos > maxPos {
			maxPos = pos
		}
		core.accounts.EXPECT().Add(gomock.Any()).Times(2)
		marginBal, generalBal := core.colE.CreateTraderAccount(row.Cells[0].Value, market, core.Asset)
		core.accounts.EXPECT().Add(gomock.Any()).Times(2)
		_ = core.colE.IncrementBalance(margin, marginBal)
		_ = core.colE.IncrementBalance(general, generalBal)
		// add trader accounts to map - this is the state they should have now
		core.traderAccs[rows.Cells[0].Value] = map[proto.AccountType]*proto.Account{
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
	market := core.GetID()
	// build + place all orders
	for _, row := range orderT.Rows {
		if row.Cells[0].Value == "trader" {
			continue
		}
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
		order := proto.Order{
			Id:        uuid.NewV4().String(),
			MarketID:  market,
			PartyID:   row.Cells[0].Value,
			Side:      side,
			Price:     uint64(price),
			Size:      uint64(vol),
			ExpiresAt: tomorrow.Unix(),
		}
		if _, err := core.SubmitOrder(&order); err != nil {
			return err
		}
	}
	return nil
}

func iCheckTheUpdatedBalancesAndPositions() error {
	return godog.ErrPending
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
	s.Step(`^the market ([A-Z\\]{7})$`, theMarket)
	s.Step(`^the system accounts:$`, theSystemAccounts)
	s.Step(`^traders have the following state:$`, tradersHaveTheFollowingState)
	s.Step(`^the following orders:$`, theFollowingOrders)
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
