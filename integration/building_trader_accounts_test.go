package core_test

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/proto"

	"github.com/DATA-DOG/godog/gherkin"
	uuid "github.com/satori/go.uuid"
)

func theExecutonEngineHaveTheseMarkets(arg1 *gherkin.DataTable) error {
	mkts := []proto.Market{}
	for _, row := range arg1.Rows {
		if val(row, 0) == "name" {
			continue
		}
		mkt := baseMarket(row)
		mkts = append(mkts, mkt)
	}

	execsetup = getExecutionTestSetup(mkts)

	return nil
}

func theFollowingTraders(arg1 *gherkin.DataTable) error {
	// create the trader from the table using NotifyTraderAccount
	for _, row := range arg1.Rows {
		if val(row, 0) == "name" {
			continue
		}

		// row.0 = traderID, row.1 = amount to topup
		notif := proto.NotifyTraderAccount{
			TraderID: val(row, 0),
			Amount:   u64val(row, 1),
		}

		err := execsetup.engine.NotifyTraderAccount(&notif)
		if err != nil {
			return err
		}

		// expected general accounts for the trader
		// added expected market margin accounts
		for _, mkt := range execsetup.mkts {
			asset, err := mkt.GetAsset()
			if err != nil {
				return err
			}

			if !traderHaveGeneralAccount(execsetup.accs[val(row, 0)], asset) {
				acc := account{
					Type:    proto.AccountType_GENERAL,
					Balance: u64val(row, 1),
					Asset:   asset,
				}
				execsetup.accs[val(row, 0)] = append(execsetup.accs[val(row, 0)], acc)
			}

			acc := account{
				Type:    proto.AccountType_MARGIN,
				Balance: 0,
				Market:  mkt.Name,
				Asset:   asset,
			}
			execsetup.accs[val(row, 0)] = append(execsetup.accs[val(row, 0)], acc)
		}

	}
	return nil
}

func iExpectTheTradersToHaveNewGeneralAccount(arg1 *gherkin.DataTable) error {
	for _, row := range arg1.Rows {
		if val(row, 0) == "name" {
			continue
		}

		_, err := execsetup.accounts.getTraderGeneralAccount(val(row, 0), val(row, 1))
		if err != nil {
			return fmt.Errorf("missing general account for trader=%v asset=%v", val(row, 0), val(row, 1))
		}
	}
	return nil
}

func generalAccountsBalanceIs(arg1, arg2 string) error {
	balance, _ := strconv.ParseUint(arg2, 10, 0)
	for _, mkt := range execsetup.mkts {
		asset, _ := mkt.GetAsset()
		acc, err := execsetup.accounts.getTraderGeneralAccount(arg1, asset)
		if err != nil {
			return err
		}
		if uint64(acc.Balance) != balance {
			return errors.New("invalid general account balance")
		}
	}
	return nil
}

func haveOnlyOneAccountPerAsset(arg1 string) error {
	assets := map[string]struct{}{}

	for _, acc := range execsetup.accounts.data {
		if acc.Owner == arg1 && acc.Type == proto.AccountType_GENERAL {
			if _, ok := assets[acc.Asset]; ok {
				return fmt.Errorf("trader=%v have multiple account for asset=%v", arg1, acc.Asset)
			}
			assets[acc.Asset] = struct{}{}
		}
	}
	return nil
}

func haveOnlyOnMarginAccountPerMarket(arg1 string) error {
	assets := map[string]struct{}{}

	for _, acc := range execsetup.accounts.data {
		fmt.Printf("acc: %v\n", acc)
		if acc.Owner == arg1 && acc.Type == proto.AccountType_MARGIN {
			if _, ok := assets[acc.MarketID]; ok {
				return fmt.Errorf("trader=%v have multiple account for market=%v", arg1, acc.MarketID)
			}
			assets[acc.MarketID] = struct{}{}
		}
	}
	return nil
}

func theMakesADepositOfIntoTheAccount(trader, amountstr, asset string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	// row.0 = traderID, row.1 = amount to topup
	notif := proto.NotifyTraderAccount{
		TraderID: trader,
		Amount:   amount,
	}

	err := execsetup.engine.NotifyTraderAccount(&notif)
	if err != nil {
		return err
	}

	return nil
}

func generalAccountForAssetBalanceIs(trader, asset, balancestr string) error {
	balance, _ := strconv.ParseUint(balancestr, 10, 0)
	acc, err := execsetup.accounts.getTraderGeneralAccount(trader, asset)
	if err != nil {
		return err
	}

	if uint64(acc.Balance) != balance {
		return fmt.Errorf("invalid general asset=%v account balance=%v for trader=%v", asset, acc.Balance, trader)
	}

	return nil
}

func theWithdrawFromTheAccount(trader, amountstr, asset string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	// row.0 = traderID, row.1 = amount to topup
	notif := proto.Withdraw{
		PartyID: trader,
		Amount:  amount,
		Asset:   asset,
	}

	err := execsetup.engine.Withdraw(&notif)
	if err != nil {
		return err
	}

	return nil

}

func tradersPlaceFollowingOrders(orders *gherkin.DataTable) error {
	for _, row := range orders.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		order := proto.Order{
			Id:          uuid.NewV4().String(),
			MarketID:    val(row, 1),
			PartyID:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   time.Now().Add(24 * time.Hour).UnixNano(),
			Type:        proto.Order_LIMIT,
			TimeInForce: proto.Order_GTT,
			CreatedAt:   time.Now().UnixNano(),
		}
		result, err := execsetup.engine.SubmitOrder(&order)
		if err != nil {
			return err
		}
		if int64(len(result.Trades)) != i64val(row, 5) {
			return fmt.Errorf("expected %d trades, instead saw %d (%#v)", i64val(row, 5), len(result.Trades), *result)
		}
	}
	return nil
}

func iExpectTheTraderToHaveAMargin(arg1 *gherkin.DataTable) error {
	for _, row := range arg1.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		account, err := execsetup.accounts.getTraderGeneralAccount(val(row, 0), val(row, 1))
		if err != nil {
			return err
		}
		if account.GetBalance() != i64val(row, 4) {
			return fmt.Errorf("expected general balance  %d, instead saw %d (trader: %v)", i64val(row, 4), account.GetBalance(), val(row, 0))
		}
		account, err = execsetup.accounts.getTraderMarginAccount(val(row, 0), val(row, 2))
		if err != nil {
			return err
		}
		if account.GetBalance() != i64val(row, 3) {
			return fmt.Errorf("expected margin balance  %d, instead saw %d (trader: %v)", i64val(row, 3), account.GetBalance(), val(row, 0))
		}
	}
	return nil
}

func baseMarket(row *gherkin.TableRow) proto.Market {
	maturity := time.Now().Add(365 * 24 * time.Hour)
	return proto.Market{
		Id:            fmt.Sprintf("%s", val(row, 0)),
		Name:          fmt.Sprintf("%s", val(row, 0)),
		DecimalPlaces: 2,
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:        fmt.Sprintf("Crypto/%s/Futures", val(row, 0)),
				Code:      fmt.Sprintf("CRYPTO/%v", val(row, 0)),
				Name:      fmt.Sprintf("%s future", val(row, 0)),
				BaseName:  val(row, 1),
				QuoteName: val(row, 2),
				Metadata: &proto.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				InitialMarkPrice: u64val(row, 4),
				Product: &proto.Instrument_Future{
					Future: &proto.Future{
						Maturity: maturity.Format("2006-01-02T15:04:05Z"),
						Oracle: &proto.Future_EthereumEvent{
							EthereumEvent: &proto.EthereumEvent{
								ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
								Event:      "price_changed",
							},
						},
						Asset: val(row, 3),
					},
				},
			},
			RiskModel: &proto.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &proto.SimpleRiskModel{
					Params: &proto.SimpleModelParams{
						FactorLong:  f64val(row, 6),
						FactorShort: f64val(row, 7),
					},
				},
			},
			MarginCalculator: &proto.MarginCalculator{
				ScalingFactors: &proto.ScalingFactors{
					SearchLevel:       f64val(row, 13),
					InitialMargin:     f64val(row, 12),
					CollateralRelease: f64val(row, 11),
				},
			},
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}

}
