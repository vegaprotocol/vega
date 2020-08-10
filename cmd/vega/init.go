package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/faucet"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/jsonpb"
	"github.com/spf13/cobra"
	"github.com/zannen/toml"
)

type initCommand struct {
	command

	rootPath              string
	force                 bool
	nodeWalletPassphrase  string
	genDevNodeWallet      bool
	genBuiltinAssetFaucet bool
	Log                   *logging.Logger
}

func (ic *initCommand) Init(c *Cli) {
	ic.cli = c
	ic.cmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize a vega node",
		Long:  "Generate the minimal configuration required for a vega node to start",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunInit(ic.rootPath, ic.force, ic.Log, ic.nodeWalletPassphrase, ic.genDevNodeWallet, ic.genBuiltinAssetFaucet)
		},
	}

	fs := ic.cmd.Flags()
	fs.StringVarP(&ic.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	fs.StringVarP(&ic.nodeWalletPassphrase, "nodewallet-passphrase", "n", "", "The path to a file containing the passphrase used to encrypt the vega nodewallet")
	fs.BoolVarP(&ic.force, "force", "f", false, "Erase exiting vega configuration at the specified path")
	fs.BoolVarP(&ic.genDevNodeWallet, "gen-dev-nodewallet", "g", false, "Generate dev wallet for all vega supported chains (not for production)")
	fs.BoolVarP(&ic.genBuiltinAssetFaucet, "gen-builtinasset-faucet", "b", false, "Generate the builtin asset configuration (not for production)")
}

// RunInit initialises vega config files - config.toml and markets/*.json.
func RunInit(rootPath string, force bool, logger *logging.Logger, nodeWalletPassphraseInput string, genDevNodeWallet, genBuiltinAssetFaucet bool) error {

	// if theses is not specified, we then trigger a prompt
	// for the user to type his password
	var (
		nodeWalletPassphrase string
		err                  error
	)
	if len(nodeWalletPassphraseInput) <= 0 {
		nodeWalletPassphrase, err = getTerminalPassphrase("nodewallet")
	} else {
		nodeWalletPassphrase, err = getFilePassphrase(nodeWalletPassphraseInput)
	}
	if err != nil {
		return fmt.Errorf("cannot start the node, passphrase error: %v", err)
	}

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
	if err = fsutil.EnsureDir(rootPath); err != nil {
		return err
	}

	fullCandleStorePath := filepath.Join(rootPath, storage.CandlesDataPath)
	fullOrderStorePath := filepath.Join(rootPath, storage.OrdersDataPath)
	fullTradeStorePath := filepath.Join(rootPath, storage.TradesDataPath)
	fullMarketStorePath := filepath.Join(rootPath, storage.MarketsDataPath)

	// create sub-folders
	if err = fsutil.EnsureDir(fullCandleStorePath); err != nil {
		return err
	}
	if err = fsutil.EnsureDir(fullOrderStorePath); err != nil {
		return err
	}
	if err = fsutil.EnsureDir(fullTradeStorePath); err != nil {
		return err
	}
	if err = fsutil.EnsureDir(fullMarketStorePath); err != nil {
		return err
	}

	// create default market folder
	fullDefaultMarketConfigPath :=
		filepath.Join(rootPath, execution.MarketConfigPath)

	if err = fsutil.EnsureDir(fullDefaultMarketConfigPath); err != nil {
		return err
	}

	// generate default market config
	filenames, err := createDefaultMarkets(fullDefaultMarketConfigPath)
	if err != nil {
		return err
	}

	// generate a default configuration
	cfg := config.NewDefaultConfig(rootPath)

	// initialize the faucet if needed
	if genBuiltinAssetFaucet {
		pubkey, err := faucet.GenConfig(logger, rootPath, nodeWalletPassphrase, false)
		if err != nil {
			return err
		}
		// add the pubkey to the whitelis
		cfg.EvtForward.BlockchainQueueWhitelist = append(
			cfg.EvtForward.BlockchainQueueWhitelist, pubkey)
	}

	// setup the defaults markets
	cfg.Execution.Markets.Configs = filenames

	// write configuration to toml
	buf := new(bytes.Buffer)
	if err = toml.NewEncoder(buf).Encode(cfg); err != nil {
		return err
	}

	// create the configuration file
	f, err := os.Create(filepath.Join(rootPath, "config.toml"))
	if err != nil {
		return err
	}

	if _, err = f.WriteString(buf.String()); err != nil {
		return err
	}

	// init the nodewallet
	if err := nodeWalletInit(cfg, nodeWalletPassphrase, genDevNodeWallet); err != nil {
		return err
	}

	// init the devAssets
	if err := assets.GenDevAssetSourcesPath(rootPath); err != nil {
		return err
	}

	logger.Info("configuration generated successfully", logging.String("path", rootPath))

	return nil
}

func nodeWalletInit(cfg config.Config, nodeWalletPassphrase string, genDevNodeWallet bool) error {
	if genDevNodeWallet {
		return nodewallet.DevInit(
			cfg.NodeWallet.StorePath,
			cfg.NodeWallet.DevWalletsPath,
			nodeWalletPassphrase,
		)
	}
	return nodewallet.Init(
		cfg.NodeWallet.StorePath,
		nodeWalletPassphrase,
	)
}

func createDefaultMarkets(confpath string) ([]string, error) {
	/*
		Notes on default markets:
		- If decimalPlaces==2, then a currency balance of `1` indicates one Euro cent, not one Euro
		- Maturity dates should be not all the same, for variety.
	*/
	skels := []struct {
		id                    string
		decimalPlaces         uint64
		baseName              string
		settlementAsset       string
		quoteName             string
		maturity              time.Time
		initialMarkPrice      uint64
		settlementValue       uint64
		sigma                 float64
		riskAversionParameter float64
	}{
		{
			id:                    "VHSRA2G5MDFKREFJ5TOAGHZBBDGCYS67",
			decimalPlaces:         5,
			baseName:              "ETH",
			quoteName:             "VUSD",
			settlementAsset:       "VUSD",
			maturity:              time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC),
			initialMarkPrice:      1410000,
			settlementValue:       1500000,
			riskAversionParameter: 0.001,
			sigma:                 1.5,
		},
		{
			id:                    "LBXRA65PN4FN5HBWRI2YBCOYDG2PBGYU",
			decimalPlaces:         5,
			baseName:              "GBP",
			quoteName:             "VUSD",
			settlementAsset:       "VUSD",
			maturity:              time.Date(2020, 10, 30, 22, 59, 59, 0, time.UTC),
			initialMarkPrice:      130000,
			settlementValue:       126000,
			riskAversionParameter: 0.01,
			sigma:                 0.09,
		},
		{
			id:                    "RTJVFCMFZZQQLLYVSXTWEN62P6AH6OCN",
			decimalPlaces:         5,
			baseName:              "ETH",
			quoteName:             "BTC",
			settlementAsset:       "BTC",
			maturity:              time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC),
			initialMarkPrice:      10000,
			settlementValue:       98123,
			riskAversionParameter: 0.001,
			sigma:                 2.0,
		},
	}

	filenames := make([]string, len(skels))

	for seq, skel := range skels {
		monYear := skel.maturity.Format("Jan06")
		monYearUpper := strings.ToUpper(monYear)

		mkt := proto.Market{
			Id:            skel.id,
			DecimalPlaces: skel.decimalPlaces,
			Fees: &proto.Fees{
				Factors: &proto.FeeFactors{
					LiquidityFee:      "0.001",
					InfrastructureFee: "0.0005",
					MakerFee:          "0.00025",
				},
			},
			TradableInstrument: &proto.TradableInstrument{
				Instrument: &proto.Instrument{
					Id:        fmt.Sprintf("Crypto/%s%s/Futures/%s", skel.baseName, skel.quoteName, monYear),
					Code:      fmt.Sprintf("CRYPTO:%s%s/%s", skel.baseName, skel.quoteName, monYearUpper),
					Name:      fmt.Sprintf("%s %s vs %s future", skel.maturity.Format("January 2006"), skel.baseName, skel.quoteName),
					BaseName:  skel.baseName,
					QuoteName: skel.quoteName,
					Metadata: &proto.InstrumentMetadata{
						Tags: []string{
							"asset_class:fx/crypto",
							"product:futures",
						},
					},
					InitialMarkPrice: skel.initialMarkPrice,
					Product: &proto.Instrument_Future{
						Future: &proto.Future{
							Maturity: skel.maturity.Format("2006-01-02T15:04:05Z"),
							Oracle: &proto.Future_EthereumEvent{
								EthereumEvent: &proto.EthereumEvent{
									ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
									Event:      "price_changed",
									Value:      skel.settlementValue,
								},
							},
							Asset: skel.settlementAsset,
						},
					},
				},
				RiskModel: &proto.TradableInstrument_LogNormalRiskModel{
					LogNormalRiskModel: &proto.LogNormalRiskModel{
						RiskAversionParameter: skel.riskAversionParameter,
						Tau:                   1.0 / 365.25 / 24,
						Params: &proto.LogNormalModelParams{
							Mu:    0,
							R:     0.016,
							Sigma: skel.sigma,
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
		filenames[seq] = fmt.Sprintf("%s%s%s.json", skel.baseName, skel.quoteName, monYearUpper)
		err := createDefaultMarket(&mkt, path.Join(confpath, filenames[seq]), uint64(seq))
		if err != nil {
			return nil, err
		}
	}

	return filenames, nil
}

func createDefaultMarket(mkt *proto.Market, path string, seq uint64) error {
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

	if _, err := f.WriteString(buf); err != nil {
		return err
	}

	return nil
}
