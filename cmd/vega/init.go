package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/fsutil"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/tomlcommentator"
	"code.vegaprotocol.io/vega/proto"

	"github.com/BurntSushi/toml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	marketBTCDEC19 = "BTCDEC19.json"
	marketETHJUN19 = "ETHJUN19.json"
	marketEURMAR19 = "EURMAR20.json"
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
	cfg, err := internal.NewDefaultConfig(ic.Log, ic.rootPath)
	if err != nil {
		return err
	}

	// setup the defaults markets
	cfg.Execution.Markets.Configs = []string{
		marketBTCDEC19, marketETHJUN19, marketEURMAR19}

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

	if _, err := f.WriteString(addTomlComments(buf.String())); err != nil {
		return err
	}

	ic.Log.Info("configuration generated successfully", zap.String("path", ic.rootPath))

	return nil
}

func addTomlComments(toml string) string {
	c := &tomlcommentator.Comments{
		Header: []string{
			"This is a TOML config file.",
			"For more information, see https://github.com/toml-lang/toml",
		},
		Items: []*tomlcommentator.CommentItem{
			&tomlcommentator.CommentItem{
				Regex: `\[Logging.Custom\]$`,
				CommentPara: []string{
					"This section takes effect only when Environment is set to \"custom\".",
				},
			},
		},
	}
	return tomlcommentator.Commentate(toml, c)
}

func createDefaultMarkets(confpath string) error {
	mkt := proto.Market{
		Id: "BTC/DEC19",
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:   "Crypto/BTCUSD/Futures/Dec19",
				Code: "FX:BTCUSD/DEC19",
				Name: "December 2019 BTC vs USD future",
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
			RiskModel: &proto.TradableInstrument_BuiltinFutures{
				BuiltinFutures: &proto.BuiltinFutures{
					HistoricVolatility: 0.15,
				},
			},
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}

	err := createDefaultMarket(&mkt, path.Join(confpath, marketBTCDEC19))
	if err != nil {
		return err
	}
	mkt.Id = "ETH/JUN19"
	mkt.TradableInstrument.Instrument.Id = "Crypto/ETHUSD/Futures/Jun19"
	mkt.TradableInstrument.Instrument.Code = "FX:ETHUSD/Jun19"
	mkt.TradableInstrument.Instrument.Name = "June 2019 ETH vs USD future"
	err = createDefaultMarket(&mkt, path.Join(confpath, marketETHJUN19))
	if err != nil {
		return err
	}
	mkt.Id = "EUR/MAR20"
	mkt.TradableInstrument.Instrument.Id = "Fx/EURUSD/Futures/Mar20"
	mkt.TradableInstrument.Instrument.Code = "FX:EURUSD/MAR20"
	mkt.TradableInstrument.Instrument.Name = "March 2020 EUR vs USD future"
	return createDefaultMarket(&mkt, path.Join(confpath, marketEURMAR19))
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
