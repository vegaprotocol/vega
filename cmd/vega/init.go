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
)

const (
	marketETHUSDDEC19 = "ETHUSDDEC19.json"
	marketGBPUSDJUN19 = "GBPUSDDEC19.json"
	marketGBPEURDEC19 = "GBPEURDEC19.json"
	//closingAt         = "2019-07-16T10:17:00Z"
	closingAt = "2019-12-31T00:00:00Z"
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
			return RunInit(ic.rootPath, ic.force, ic.Log)
		},
	}

	fs := ic.cmd.Flags()
	fs.StringVarP(&ic.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	fs.BoolVarP(&ic.force, "force", "f", false, "Erase exiting vega configuration at the specified path")

}

func RunInit(rootPath string, force bool, logger *logging.Logger) error {
	rootPathExists, err := fsutil.PathExists(rootPath)
	if err != nil {
		if _, ok := err.(*fsutil.PathNotFound); !ok {
			return err
		}
	}

	if rootPathExists && !force {
		return fmt.Errorf("configuration already exists at `%v` please remove it first or re-run using -f", rootPath)
	}

	if rootPathExists && force {
		logger.Info("removing existing configuration", logging.String("path", rootPath))
		os.RemoveAll(rootPath) // ignore any errors here to force removal
	}

	// create the root
	if err := fsutil.EnsureDir(rootPath); err != nil {
		return err
	}

	fullCandleStorePath := filepath.Join(rootPath, storage.CandlesDataPath)
	fullOrderStorePath := filepath.Join(rootPath, storage.OrdersDataPath)
	fullTradeStorePath := filepath.Join(rootPath, storage.TradesDataPath)
	fullMarketStorePath := filepath.Join(rootPath, storage.MarketsDataPath)

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
		filepath.Join(rootPath, execution.MarketConfigPath)

	if err := fsutil.EnsureDir(fullDefaultMarketConfigPath); err != nil {
		return err
	}

	// generate default market config
	err = createDefaultMarkets(fullDefaultMarketConfigPath)
	if err != nil {
		return err
	}

	// generate a default configuration
	cfg := config.NewDefaultConfig(rootPath)

	// setup the defaults markets
	cfg.Execution.Markets.Configs = []string{
		marketETHUSDDEC19, marketGBPUSDJUN19, marketGBPEURDEC19}

	// write configuration to toml
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(cfg); err != nil {
		return err
	}

	// create the configuration file
	f, err := os.Create(filepath.Join(rootPath, "config.toml"))
	if err != nil {
		return err
	}

	if _, err := f.WriteString(buf.String()); err != nil {
		return err
	}

	logger.Info("configuration generated successfully", logging.String("path", rootPath))

	return nil
}

func createDefaultMarkets(confpath string) error {
	var seq uint64
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
		Name: "ETHUSD/DEC19",
		// A currency balance of `1` indicates one US cent, not one US dollar
		DecimalPlaces: 2,
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:        "Crypto/ETHUSD/Futures/Dec19",
				Code:      "CRYPTO:ETHUSD/DEC19",
				Name:      "December 2019 ETH vs USD future",
				BaseName:  "ETH",
				QuoteName: "USD",
				Metadata: &proto.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &proto.Instrument_Future{
					Future: &proto.Future{
						Maturity: closingAt,
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
			RiskModel: riskModel,
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}

	err := createDefaultMarket(&mkt, path.Join(confpath, marketETHUSDDEC19), seq)
	if err != nil {
		return err
	}
	seq++

	mkt.Name = "GBPUSD/JUN19"
	// A currency balance of `1` indicates one US cent, not one US dollar
	mkt.DecimalPlaces = 2
	mkt.TradableInstrument.Instrument.Id = "FX/GBPUSD/Futures/Jun19"
	mkt.TradableInstrument.Instrument.Code = "FX:GBPUSD/Jun19"
	mkt.TradableInstrument.Instrument.Name = "December 2019 GBP vs USD future"
	mkt.TradableInstrument.Instrument.BaseName = "GBP"
	mkt.TradableInstrument.Instrument.Product = &proto.Instrument_Future{
		Future: &proto.Future{
			Maturity: closingAt,
			Oracle: &proto.Future_EthereumEvent{
				EthereumEvent: &proto.EthereumEvent{
					ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
					Event:      "price_changed",
				},
			},
			Asset: "USD",
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
	err = createDefaultMarket(&mkt, path.Join(confpath, marketGBPUSDJUN19), seq)
	if err != nil {
		return err
	}
	seq++

	mkt.Name = "GBPEUR/DEC19"
	// A currency balance of `1` indicates one Euro cent, not one Euro
	mkt.DecimalPlaces = 2
	mkt.TradableInstrument.Instrument.Id = "Fx/GBPEUR/Futures/Dec20"
	mkt.TradableInstrument.Instrument.Code = "FX:GBPEUR/DEC20"
	mkt.TradableInstrument.Instrument.Name = "December 2019 GBP vs EUR future"
	mkt.TradableInstrument.Instrument.BaseName = "GBP"
	mkt.TradableInstrument.Instrument.QuoteName = "EUR"
	mkt.TradableInstrument.Instrument.Product = &proto.Instrument_Future{
		Future: &proto.Future{
			Maturity: closingAt,
			Oracle: &proto.Future_EthereumEvent{
				EthereumEvent: &proto.EthereumEvent{
					ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
					Event:      "price_changed",
				},
			},
			Asset: "EUR",
		},
	}
	return createDefaultMarket(&mkt, path.Join(confpath, marketGBPEURDEC19), seq)
}

func createDefaultMarket(mkt *proto.Market, path string, seq uint64) error {
	execution.SetMarketID(mkt, seq)
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
