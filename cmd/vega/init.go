package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"code.vegaprotocol.io/vega/internal/config"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/fsutil"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/jsonpb"
	"github.com/spf13/cobra"
	"github.com/zannen/toml"
	"go.uber.org/zap"
)

const (
	marketETHDEC19 = "ETHDEC19.json"
	marketGBPJUN19 = "GBPJUN19.json"
	marketBTCDEC19 = "BTCDEC19.json"
)

type initCommand struct {
	command

	rootPath string
	force    bool
	Log      *logging.Logger
}

func (ic *initCommand) Init(c *Cli) {
	ic.cli = c
	ic.cmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize a vega node",
		Long:  "Generate the minimal configuration required for a vega node to start",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ic.runInit(c)
		},
	}

	fs := ic.cmd.Flags()
	fs.StringVarP(&ic.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	fs.BoolVarP(&ic.force, "force", "f", false, "Erase exiting vega configuration at the specified path")

}

func (ic *initCommand) runInit(c *Cli) error {
	rootPathExists, err := fsutil.PathExists(ic.rootPath)
	if err != nil {
		if _, ok := err.(*fsutil.PathNotFound); !ok {
			return err
		}
	}

	if rootPathExists && !ic.force {
		return fmt.Errorf("configuration already exists at `%v` please remove it first or re-run using -f", ic.rootPath)
	}

	if rootPathExists && ic.force {
		ic.Log.Info("removing existing configuration", zap.String("path", ic.rootPath))
		os.RemoveAll(ic.rootPath) // ignore any errors here to force removal
	}

	// create the root
	if err := fsutil.EnsureDir(ic.rootPath); err != nil {
		return err
	}

	fullCandleStorePath := filepath.Join(ic.rootPath, storage.CandleStoreDataPath)
	fullOrderStorePath := filepath.Join(ic.rootPath, storage.OrderStoreDataPath)
	fullTradeStorePath := filepath.Join(ic.rootPath, storage.TradeStoreDataPath)
	fullMarketStorePath := filepath.Join(ic.rootPath, storage.MarketStoreDataPath)

	// create sub-folders
	if err := fsutil.EnsureDir(fullCandleStorePath); err != nil {
		return err
	}
	if err := fsutil.EnsureDir(fullOrderStorePath); err != nil {
		return err
	}
	if err := fsutil.EnsureDir(fullTradeStorePath); err != nil {
		return err
	}
	if err := fsutil.EnsureDir(fullMarketStorePath); err != nil {
		return err
	}

	// create default market folder
	fullDefaultMarketConfigPath :=
		filepath.Join(ic.rootPath, execution.MarketConfigPath)

	if err := fsutil.EnsureDir(fullDefaultMarketConfigPath); err != nil {
		return err
	}

	// generate default market config
	err = createDefaultMarkets(fullDefaultMarketConfigPath)
	if err != nil {
		return err
	}

	// generate a default configuration
	cfg := config.NewDefaultConfig(ic.rootPath)

	// setup the defaults markets
	cfg.Execution.Markets.Configs = []string{
		marketETHDEC19, marketGBPJUN19, marketBTCDEC19}

	// write configuration to toml
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(cfg); err != nil {
		return err
	}

	// create the configuration file
	f, err := os.Create(filepath.Join(ic.rootPath, "config.toml"))
	if err != nil {
		return err
	}

	if _, err := f.WriteString(buf.String()); err != nil {
		return err
	}

	ic.Log.Info("configuration generated successfully", zap.String("path", ic.rootPath))

	return nil
}

func createDefaultMarkets(confpath string) error {
	riskModel := &proto.TradableInstrument_Forward{
		Forward: &proto.Forward{
			Lambd: 0.01,
			Tau:   1.0 / 365.25 / 24,
			Params: &proto.ModelParamsBS{
				Mu:    0,
				R:     0.016,
				Sigma: 0.09,
			},
		},
	}

	mkt := proto.Market{
		Id: "ETH/DEC19",
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:   "Crypto/ETHUSD/Futures/Dec19",
				Code: "FX:BTCUSD/DEC19",
				Name: "December 2019 ETH vs USD future",
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
						Asset: "Ethereum/Ether",
					},
				},
			},
			RiskModel: riskModel,
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}

	err := createDefaultMarket(&mkt, path.Join(confpath, marketETHDEC19))
	if err != nil {
		return err
	}
	mkt.Id = "GBP/JUN19"
	mkt.TradableInstrument.Instrument.Id = "FX/GBPUSD/Futures/Jun19"
	mkt.TradableInstrument.Instrument.Code = "FX:GBPUSD/Jun19"
	mkt.TradableInstrument.Instrument.Name = "June 2019 GBP vs USD future"
	mkt.TradableInstrument.Instrument.Product = &proto.Instrument_Future{
		Future: &proto.Future{
			Maturity: "2019-06-30T00:00:00Z",
			Oracle: &proto.Future_EthereumEvent{
				EthereumEvent: &proto.EthereumEvent{
					ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
					Event:      "price_changed",
				},
			},
			Asset: "Ethereum/Ether",
		},
	}
	mkt.TradableInstrument.RiskModel = &proto.TradableInstrument_Forward{
		Forward: &proto.Forward{
			Lambd: 0.01,
			Tau:   1.0 / 365.25 / 24,
			Params: &proto.ModelParamsBS{
				Mu:    0,
				R:     0.0,
				Sigma: 0.9,
			},
		},
	}
	err = createDefaultMarket(&mkt, path.Join(confpath, marketGBPJUN19))
	if err != nil {
		return err
	}
	mkt.Id = "BTC/DEC19"
	mkt.TradableInstrument.Instrument.Id = "Fx/BTCUSD/Futures/Mar20"
	mkt.TradableInstrument.Instrument.Code = "FX:BTCUSD/MAR20"
	mkt.TradableInstrument.Instrument.Name = "DEC 2019 BTC vs USD future"
	mkt.TradableInstrument.Instrument.Product = &proto.Instrument_Future{
		Future: &proto.Future{
			Maturity: "2019-12-31T00:00:00Z",
			Oracle: &proto.Future_EthereumEvent{
				EthereumEvent: &proto.EthereumEvent{
					ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
					Event:      "price_changed",
				},
			},
			Asset: "Ethereum/Ether",
		},
	}
	return createDefaultMarket(&mkt, path.Join(confpath, marketBTCDEC19))
}

func createDefaultMarket(mkt *proto.Market, path string) error {
	m := jsonpb.Marshaler{
		Indent: "  ",
	}
	buf, err := m.MarshalToString(mkt)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if _, err := f.WriteString(string(buf)); err != nil {
		return err
	}

	return nil
}
