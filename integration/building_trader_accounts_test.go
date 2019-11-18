package core_test

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/proto"

	"github.com/DATA-DOG/godog/gherkin"
)

func theExecutonEngineHaveTheseMarkets(arg1 *gherkin.DataTable) error {
	mkts := []proto.Market{}
	for _, row := range arg1.Rows {
		if val(row, 0) == "name" {
			continue
		}
		mkt := baseMarket(val(row, 0), val(row, 1), val(row, 2), val(row, 3))
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

func baseMarket(name, baseName, quoteName, asset string) proto.Market {
	maturity := time.Now().Add(365 * 24 * time.Hour)
	return proto.Market{
		Id:            fmt.Sprintf("%s", name),
		Name:          fmt.Sprintf("%s", name),
		DecimalPlaces: 2,
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:        fmt.Sprintf("Crypto/%s/Futures", name),
				Code:      fmt.Sprintf("CRYPTO/%v", name),
				Name:      fmt.Sprintf("%s future", name),
				BaseName:  baseName,
				QuoteName: quoteName,
				Metadata: &proto.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				InitialMarkPrice: 1000,
				Product: &proto.Instrument_Future{
					Future: &proto.Future{
						Maturity: maturity.Format("2006-01-02T15:04:05Z"),
						Oracle: &proto.Future_EthereumEvent{
							EthereumEvent: &proto.EthereumEvent{
								ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
								Event:      "price_changed",
							},
						},
						Asset: asset,
					},
				},
			},
			RiskModel: &proto.TradableInstrument_ForwardRiskModel{
				ForwardRiskModel: &proto.ForwardRiskModel{
					RiskAversionParameter: 0.01,
					Tau:                   1.0 / 365.25 / 24,
					Params: &proto.ModelParamsBS{
						Mu:    0,
						R:     0.016,
						Sigma: 0.09,
					},
				},
			},
			MarginCalculator: &proto.MarginCalculator{
				ScalingFactors: &proto.ScalingFactors{
					SearchLevel:       1.1,
					InitialMargin:     1.2,
					CollateralRelease: 1.4,
				},
			},
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}

}
