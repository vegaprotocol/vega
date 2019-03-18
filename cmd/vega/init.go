package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/proto"

	"github.com/BurntSushi/toml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
	fs.StringVarP(&ic.rootPath, "root-path", "r", defaultVegaDir(), "Path of the root directory in which the configuration will be located")
	fs.BoolVarP(&ic.force, "force", "f", false, "Erase exiting vega configuration at the specified path")

}

func (ic *initCommand) runInit(c *Cli) error {
	rootPathExists, err := pathExists(ic.rootPath)
	if err != nil {
		if _, ok := err.(*PathNotFound); !ok {
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
	if err := ensureDir(ic.rootPath); err != nil {
		return err
	}

	fullCandleStorePath := filepath.Join(ic.rootPath, storage.CandleStoreDataPath)
	fullOrderStorePath := filepath.Join(ic.rootPath, storage.OrderStoreDataPath)
	fullTradeStorePath := filepath.Join(ic.rootPath, storage.TradeStoreDataPath)
	fullMarketStorePath := filepath.Join(ic.rootPath, storage.MarketStoreDataPath)

	// create sub-folders
	if err := ensureDir(fullCandleStorePath); err != nil {
		return err
	}
	if err := ensureDir(fullOrderStorePath); err != nil {
		return err
	}
	if err := ensureDir(fullTradeStorePath); err != nil {
		return err
	}
	if err := ensureDir(fullMarketStorePath); err != nil {
		return err
	}

	// create default market folder
	fullDefaultMarketConfigPath :=
		filepath.Join(ic.rootPath, execution.MarketConfigPath)

	if err := ensureDir(fullDefaultMarketConfigPath); err != nil {
		return err
	}

	// generate default market config
	err = createDefaultMarket(
		filepath.Join(
			fullDefaultMarketConfigPath,
			execution.DefaultMarketConfigName))
	if err != nil {
		return err
	}

	// generate a default configuration
	cfg, err := internal.NewDefaultConfig(ic.Log, ic.rootPath)
	if err != nil {
		return err
	}

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

func createDefaultMarket(confpath string) error {
	defaultMarket := proto.Market{
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
						Maturity: "2019-12-31",
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

	m := jsonpb.Marshaler{
		Indent: "  ",
	}
	buf, err := m.MarshalToString(&defaultMarket)
	if err != nil {
		return err
	}

	f, err := os.Create(confpath)
	if err != nil {
		return err
	}

	if _, err := f.WriteString(string(buf)); err != nil {
		return err
	}

	return nil
}
