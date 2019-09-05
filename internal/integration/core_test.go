package core_test

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/internal/collateral"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/matching"
	"code.vegaprotocol.io/vega/internal/positions"
	"code.vegaprotocol.io/vega/internal/risk"
	"code.vegaprotocol.io/vega/internal/settlement"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/DATA-DOG/godog"
	_ "github.com/DATA-DOG/godog/cmd/godog"
	"github.com/DATA-DOG/godog/gherkin"
	uuid "github.com/satori/go.uuid"
)

type traderState struct {
	pos             int
	margin, general int64
	markPrice       int
	gAcc            *types.Account
	mAcc            *types.Account
}

var (
	core     *execution.Market
	accounts *storage.Account
)

func theMarket(market string) error {
	parts := strings.Split(market, "/")
	mkt := &types.Market{
		Id: market,
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				Id:   fmt.Sprintf("Crypto/%s/Futures/%s", parts[0], parts[1]),
				Code: fmt.Sprintf("FX:%s%s", parts[0], parts[1]),
				Name: "December 2019 test future",
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &types.Instrument_Future{
					Future: &types.Future{
						Maturity: "2019-12-31T00:00:00Z",
						Oracle: &types.Future_EthereumEvent{
							EthereumEvent: &types.EthereumEvent{
								ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
								Event:      "price_changed",
							},
						},
						Asset: "Ethereum/Ether",
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
		TradingMode: &types.Market_Continuous{
			Continuous: &types.ContinuousTrading{},
		},
	}
	log := logging.NewTestLogger()
	storageConf := storage.NewDefaultConfig("/tmp")
	candles, err := storage.NewCandles(log, storageConf)
	if err != nil {
		return err
	}
	orders, err := storage.NewOrders(log, storageConf, func() {})
	if err != nil {
		return err
	}
	trades, err := storage.NewTrades(log, storageConf, func() {})
	if err != nil {
		return err
	}
	// these New funcs can't fail
	parties, _ := storage.NewParties(storageConf)
	accounts, _ = storage.NewAccounts(log, storageConf)
	m, err := execution.NewMarket(
		log,
		risk.NewDefaultConfig(),
		collateral.NewDefaultConfig(),
		positions.NewDefaultConfig(),
		settlement.NewDefaultConfig(),
		matching.NewDefaultConfig(),
		mkt,
		candles,
		orders,
		parties,
		trades,
		accounts,
		time.Now(),
		1, // seq?
	)
	if err != nil {
		return err
	}
	core = m
	return nil
}

func theSystemAccounts(systemAccounts *gherkin.DataTable) error {
	accTypes := map[string]types.AccountType{
		"settlement": types.AccountType_SETTLEMENT,
		"insurance":  types.AccountType_INSURANCE,
	}
	// set system account balances
	for _, row := range systemAccounts.Rows {
		if accT, ok := accTypes[row.Cells[0].Value]; ok {
			sacc, err := accounts.GetAccountsForOwnerByType(storage.SystemOwner, accT)
			if err != nil {
				return err
			}
			bal, err := strconv.ParseInt(row.Cells[2].Value, 10, 64)
			if err != nil {
				return err
			}
			// system will only have 1 account here
			if err := accounts.UpdateBalance(sacc[0].Id, bal); err != nil {
				return err
			}
		}
	}
	return nil
}

func tradersHaveTheFollowingState(traders *gherkin.DataTable) error {
	// damn... positions engine is not open here, let's just ram through the trades, and update the balances after the fact
	market := core.GetID()
	maxPos := 100 // ensure we can move 100 positions either long or short, doesn't really matter which way
	traderStates := map[string]traderState{}
	tomorrow := time.Now().Add(time.Hour * 24)
	// each position will be put down as an order
	orders := make([]*types.Order, 0, len(traders.Rows))
	for _, row := range traders.Rows {
		// skip first row
		if row.Cells[0].Value == "trader" {
			continue
		}
		// it's safe to ignore this error for now
		_ = accounts.CreateTraderMarketAccounts(row.Cells[0].Value, market)
		pos, err := strconv.Atoi(row.Cells[1].Value)
		if err != nil {
			return err
		}
		margin, err := strconv.ParseInt(row.Cells[2].Value, 10, 64)
		if err != nil {
			return err
		}
		general, err := strconv.ParseInt(row.Cells[3].Value, 10, 64)
		if err != nil {
			return err
		}
		mark, err := strconv.Atoi(row.Cells[5].Value)
		if err != nil {
			return err
		}
		// highest net pos
		if pos > maxPos {
			maxPos = pos
		}
		ts := traderState{
			general:   general,
			margin:    margin,
			markPrice: mark,
			pos:       pos,
		}
		// make sure there's ample margin balance to get to the positions we need
		// get accounts:
		gen, err := accounts.GetAccountsForOwnerByType(row.Cells[0].Value, types.AccountType_GENERAL)
		if err != nil {
			return err
		}
		mAcc, err := accounts.GetAccountsForOwnerByType(row.Cells[0].Value, types.AccountType_MARGIN)
		if err != nil {
			return err
		}
		// there's only 1 account for these owners, we've just set them up
		if err := accounts.UpdateBalance(gen[0].Id, int64(100*maxPos)); err != nil {
			return err
		}
		if err := accounts.UpdateBalance(mAcc[0].Id, int64(10*maxPos)); err != nil {
			return err
		}
		ts.gAcc = gen[0]
		ts.mAcc = mAcc[0]
		// add to states
		side := types.Side_Buy
		vol := ts.pos
		if pos < 0 {
			side = types.Side_Sell
			// absolute value for volume
			vol *= -1
		}
		traderStates[row.Cells[0].Value] = ts
		order := types.Order{
			Id:        uuid.NewV4().String(),
			MarketID:  market,
			PartyID:   row.Cells[0].Value,
			Side:      side,
			Price:     1,
			Size:      uint64(vol),
			ExpiresAt: tomorrow.Unix(),
		}
		// get order ready to submit
		orders = append(orders, &order)
	}
	// ok, submit some 'fake' orders, ensuring that the traders' positions all match up
	for _, o := range orders {
		if _, err := core.SubmitOrder(o); err != nil {
			return err
		}
	}
	// update their account balances, so we have established the traders' states
	for _, ts := range traderStates {
		if err := accounts.UpdateBalance(ts.gAcc.Id, ts.general); err != nil {
			return err
		}
		if err := accounts.UpdateBalance(ts.mAcc.Id, ts.margin); err != nil {
			return err
		}
	}
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
		side := types.Side_Buy
		if row.Cells[1].Value == "sell" {
			side = types.Side_Sell
		}
		vol, err := strconv.Atoi(row.Cells[2].Value)
		if err != nil {
			return err
		}
		price, err := strconv.ParseInt(row.Cells[3].Value, 10, 64)
		if err != nil {
			return err
		}
		order := types.Order{
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
	s.Step(`^the market ([A-Z\\]{7})$`, theMarket)
	s.Step(`^the system accounts:$`, theSystemAccounts)
	s.Step(`^traders have the following state:$`, tradersHaveTheFollowingState)
	s.Step(`^the following orders:$`, theFollowingOrders)
	s.Step(`^I check the updated balances and positions$`, iCheckTheUpdatedBalancesAndPositions)
	s.Step(`^I expect to see:$`, iExpectToSee)
}
