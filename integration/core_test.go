package core_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	// cmocks "code.vegaprotocol.io/vega/collateral/mocks"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"

	"github.com/cucumber/godog/gherkin"
	uuid "github.com/satori/go.uuid"
)

var (
	core *execution.Market
)

func initialiseMarket(row *gherkin.TableRow, mkt *proto.Market) {
	// the header of the feature file (ie where to find the data in the row) looks like this:
	// | name      | markprice | risk model | lamd | tau         | mu | r | sigma     | release factor | initial factor | search factor |

	// general stuff like name, ID, code, asset, and initial mark price
	mkt.Name = row.Cells[0].Value
	parts := strings.Split(mkt.Name, "/")
	mkt.Id = fmt.Sprintf("Crypto/%s/Futures/%s", parts[0], parts[1])
	mkt.TradableInstrument.Instrument.Code = fmt.Sprintf("FX:%s%s", parts[0], parts[1])
	prod := mkt.TradableInstrument.Instrument.GetFuture()
	prod.Asset = parts[0]
	mkt.TradableInstrument.Instrument.Product = &proto.Instrument_Future{
		Future: prod,
	} // set asset, reassign the product
	mkt.TradableInstrument.Instrument.InitialMarkPrice, _ = strconv.ParseUint(row.Cells[1].Value, 10, 64)

	// whether it's lambd/tau or short/long depends on the risk model
	lambdShort, _ := strconv.ParseFloat(row.Cells[3].Value, 64)
	tauLong, _ := strconv.ParseFloat(row.Cells[4].Value, 64)
	// we'll always need to use these
	release, _ := strconv.ParseFloat(row.Cells[8].Value, 64)
	initial, _ := strconv.ParseFloat(row.Cells[9].Value, 64)
	search, _ := strconv.ParseFloat(row.Cells[10].Value, 64)

	// set scaling factors
	mkt.TradableInstrument.MarginCalculator.ScalingFactors = &proto.ScalingFactors{
		SearchLevel:       search,
		InitialMargin:     initial,
		CollateralRelease: release,
	}

	// simple risk model:
	if row.Cells[2].Value == "simple" {
		mkt.TradableInstrument.RiskModel = &proto.TradableInstrument_SimpleRiskModel{
			SimpleRiskModel: &proto.SimpleRiskModel{
				Params: &proto.SimpleModelParams{
					FactorLong:  tauLong,
					FactorShort: lambdShort,
				},
			},
		}
		return
	}
	// for now, default to/assume future (forward risk model)
	mu, _ := strconv.ParseFloat(row.Cells[5].Value, 64)
	r, _ := strconv.ParseFloat(row.Cells[6].Value, 64)
	sigma, _ := strconv.ParseFloat(row.Cells[7].Value, 64)
	mkt.TradableInstrument.RiskModel = &proto.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &proto.LogNormalRiskModel{
			RiskAversionParameter: lambdShort,
			Tau:                   tauLong,
			Params: &proto.LogNormalModelParams{
				Mu:    mu,
				R:     r,
				Sigma: sigma,
			},
		},
	}
}

func theMarket(mSetup *gherkin.DataTable) error {
	// generic market config, ready to be populated with specs from scenario
	mkt := &proto.Market{
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
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
					},
				},
			},
			MarginCalculator: &proto.MarginCalculator{},
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}
	for _, row := range mSetup.Rows {
		// skip header
		if row.Cells[0].Value == "name" {
			continue
		}
		initialiseMarket(row, mkt)
	}
	log := logging.NewTestLogger()
	// the controller needs the reporter to report on errors or clunk out with fatal
	mktsetup = getMarketTestSetup(mkt)
	// create the party engine, and add to the test setup
	// so we can register parties and their account balances
	mktsetup.party = execution.NewParty(log, mktsetup.colE, []proto.Market{*mkt}, mktsetup.parties)
	m, err := execution.NewMarket(
		log,
		risk.NewDefaultConfig(),
		positions.NewDefaultConfig(),
		settlement.NewDefaultConfig(),
		matching.NewDefaultConfig(),
		mktsetup.colE,
		mktsetup.party, // party-engine here!
		mkt,
		mktsetup.candles,
		mktsetup.orders,
		mktsetup.parties,
		mktsetup.trades,
		mktsetup.marginLevelsBuf,
		NewSettlementStub(),
		time.Now(),
		mktsetup.broker,
		execution.NewIDGen(),
	)
	if err != nil {
		return err
	}
	mktsetup.core = m
	core = m
	return nil
}

func theSystemAccounts(systemAccounts *gherkin.DataTable) error {
	// we currently have N accounts, creating system accounts should create 2 more accounts
	current := len(mktsetup.accounts.data)
	// this should create market accounts, currently same way it's done in execution engine (register market)
	asset, _ := mktsetup.market.GetAsset()
	_, _ = mktsetup.colE.CreateMarketAccounts(mktsetup.core.GetID(), asset, 0)
	if len(mktsetup.accounts.data) != current+2 {
		reporter.err = fmt.Errorf("error creating system accounts")
	}
	return reporter.err
}

func tradersHaveTheFollowingState(traders *gherkin.DataTable) error {
	// this is going to be tricky... we have no product set up, we can only ensure the trader accounts are created,  but that's about it...
	// damn... positions engine is not open here, let's just ram through the trades, and update the balances after the fact
	market := core.GetID()
	maxPos := 100 // ensure we can move 100 positions either long or short, doesn't really matter which way
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
		marginBal, err := strconv.ParseUint(row.Cells[2].Value, 10, 64)
		if err != nil {
			return err
		}
		generalBal, err := strconv.ParseUint(row.Cells[3].Value, 10, 64)
		if err != nil {
			return err
		}
		// highest net pos
		if pos > maxPos {
			maxPos = pos
		}
		asset, _ := mktsetup.market.GetAsset()
		// get the account balance, ensure we can set the margin balance in this step if we want to
		// and get the account ID's so we can keep track of the state correctly
		general := mktsetup.colE.CreatePartyGeneralAccount(row.Cells[0].Value, asset)
		margin, _ := mktsetup.colE.CreatePartyMarginAccount(row.Cells[0].Value, market, asset)
		_ = mktsetup.colE.IncrementBalance(margin, marginBal)
		// add trader accounts to map - this is the state they should have now
		mktsetup.traderAccs[row.Cells[0].Value] = map[proto.AccountType]*proto.Account{
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
		notif := &proto.NotifyTraderAccount{
			TraderID: row.Cells[0].Value,
			Amount:   uint64(generalBal),
		}
		// we should be able to safely ignore the error, if this fails, the tests will
		_ = mktsetup.party.NotifyTraderAccountWithTopUpAmount(notif, generalBal)
	}
	return nil
}

func theFollowingOrders(orderT *gherkin.DataTable) error {
	tomorrow := time.Now().Add(time.Hour * 24)
	core := mktsetup.core
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
		// setup.parties.EXPECT().GetByID(row.Cells[0].Value).Times(1).Return(
		// 	&proto.Party{
		// 		Id: row.Cells[0].Value,
		// 	},
		// 	nil,
		// )

		side := proto.Side_SIDE_BUY
		if row.Cells[1].Value == "sell" {
			side = proto.Side_SIDE_SELL
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
			Type:        proto.Order_TYPE_LIMIT,
			TimeInForce: proto.Order_TIF_GTT,
			CreatedAt:   time.Now().UnixNano(),
		}
		result, err := core.SubmitOrder(context.TODO(), &order)
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
		margin, err := strconv.ParseUint(row.Cells[4].Value, 10, 64)
		if err != nil {
			return err
		}
		general, err := strconv.ParseUint(row.Cells[5].Value, 10, 64)
		if err != nil {
			return err
		}
		accounts := mktsetup.traderAccs[trader]
		acc, err := mktsetup.colE.GetAccountByID(accounts[proto.AccountType_MARGIN].Id)
		if err != nil {
			return err
		}
		// sync margin account state
		mktsetup.traderAccs[trader][proto.AccountType_MARGIN] = acc
		if acc.Balance != margin {
			return fmt.Errorf("expected %s margin account balance to be %d instead saw %d", trader, margin, acc.Balance)
		}
		acc, err = mktsetup.colE.GetAccountByID(accounts[proto.AccountType_GENERAL].Id)
		if err != nil {
			return err
		}
		if acc.Balance != general {
			return fmt.Errorf("expected %s general account balance to be %d, instead saw %d", trader, general, acc.Balance)
		}
		// sync general account state
		mktsetup.traderAccs[trader][proto.AccountType_GENERAL] = acc
	}
	return nil
}

func hasNotBeenAddedToTheMarket(trader string) error {
	accounts := mktsetup.traderAccs[trader]
	acc, err := mktsetup.colE.GetAccountByID(accounts[proto.AccountType_MARGIN].Id)
	if err != nil || acc.Balance == 0 {
		return nil
	}
	return fmt.Errorf("didn't expect %s to hava a margin account with balance, instead saw %d", trader, acc.Balance)
}

func theMarkPriceIs(markPrice string) error {
	price, _ := strconv.ParseUint(markPrice, 10, 64)
	marketMarkPrice := mktsetup.core.GetMarketData().MarkPrice
	if marketMarkPrice != price {
		return fmt.Errorf("expected mark price of %d instead saw %d", price, marketMarkPrice)
	}

	return nil
}
