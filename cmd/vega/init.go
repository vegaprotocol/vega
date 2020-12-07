package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/faucet"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"
	"github.com/golang/protobuf/jsonpb"
	"github.com/jessevdk/go-flags"
	"github.com/zannen/toml"
)

type InitCmd struct {
	config.RootPathFlag

	// We've unified the passphrase flag as config.PassphraseFlag, which uses --passphrase.
	// As systemtests uses --nodewallet-passphrase we'll define the flag directly here
	// TODO: uncomment this line and remove the Passphrase field.
	// config.PassphraseFlag
	Passphrase config.Passphrase `short:"p" long:"nodewallet-passphrase" description:"A file containing the passphrase for the wallet, if empty will prompt for input"`

	Force      bool `short:"f" long:"force" description:"Erase exiting vega configuration at the specified path"`
	GenDev     bool `short:"g" long:"gen-dev-nodewallet" description:"Generate dev wallet for all vega supported chains (not for production)"`
	GenBuiltin bool `short:"b" long:"gen-builtinasset-faucet" description:"Generate the builtin asset configuration (not for production)"`
}

var initCmd InitCmd

func (opts *InitCmd) Execute(_ []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	rootPathExists, err := fsutil.PathExists(opts.RootPath)
	if err != nil {
		if _, ok := err.(*fsutil.PathNotFound); !ok {
			return err
		}
	}

	if rootPathExists && !opts.Force {
		return fmt.Errorf("configuration already exists at `%v` please remove it first or re-run using -f", opts.RootPath)
	}

	if rootPathExists && opts.Force {
		logger.Info("removing existing configuration", logging.String("path", opts.RootPath))
		os.RemoveAll(opts.RootPath) // ignore any errors here to force removal
	}

	// create the root
	if err = fsutil.EnsureDir(opts.RootPath); err != nil {
		return err
	}

	fullCandleStorePath := filepath.Join(opts.RootPath, storage.CandlesDataPath)
	fullOrderStorePath := filepath.Join(opts.RootPath, storage.OrdersDataPath)
	fullTradeStorePath := filepath.Join(opts.RootPath, storage.TradesDataPath)
	fullMarketStorePath := filepath.Join(opts.RootPath, storage.MarketsDataPath)

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
		filepath.Join(opts.RootPath, execution.MarketConfigPath)

	if err = fsutil.EnsureDir(fullDefaultMarketConfigPath); err != nil {
		return err
	}

	// generate default market config
	filenames, err := createDefaultMarkets(fullDefaultMarketConfigPath)
	if err != nil {
		return err
	}

	// generate a default configuration
	cfg := config.NewDefaultConfig(opts.RootPath)

	pass, err := opts.Passphrase.Get("nodewallet")
	if err != nil {
		return err
	}

	// initialize the faucet if needed
	if opts.GenBuiltin {
		pubkey, err := faucet.GenConfig(logger, opts.RootPath, pass, false)
		if err != nil {
			return err
		}
		// add the pubkey to the allowlist
		cfg.EvtForward.BlockchainQueueAllowlist = append(
			cfg.EvtForward.BlockchainQueueAllowlist, pubkey)
	}

	// setup the defaults markets
	cfg.Execution.Markets.Configs = filenames

	// write configuration to toml
	buf := new(bytes.Buffer)
	if err = toml.NewEncoder(buf).Encode(cfg); err != nil {
		return err
	}

	// create the configuration file
	f, err := os.Create(filepath.Join(opts.RootPath, "config.toml"))
	if err != nil {
		return err
	}

	if _, err = f.WriteString(buf.String()); err != nil {
		return err
	}

	// init the nodewallet
	if err := nodeWalletInit(cfg, pass, opts.GenDev); err != nil {
		return err
	}

	logger.Info("configuration generated successfully", logging.String("path", opts.RootPath))

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
		id                     string
		decimalPlaces          uint64
		baseName               string
		settlementAsset        string
		quoteName              string
		maturity               time.Time
		initialMarkPrice       uint64
		settlementValue        uint64
		sigma                  float64
		riskAversionParameter  float64
		openingAuctionDuration string
	}{
		{
			id:                     "VHSRA2G5MDFKREFJ5TOAGHZBBDGCYS67",
			decimalPlaces:          5,
			baseName:               "ETH",
			quoteName:              "VUSD",
			settlementAsset:        "VUSD",
			maturity:               time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC),
			initialMarkPrice:       1410000,
			settlementValue:        1500000,
			riskAversionParameter:  0.001,
			sigma:                  1.5,
			openingAuctionDuration: "10s",
		},
		{
			id:                     "LBXRA65PN4FN5HBWRI2YBCOYDG2PBGYU",
			decimalPlaces:          5,
			baseName:               "GBP",
			quoteName:              "VUSD",
			settlementAsset:        "VUSD",
			maturity:               time.Date(2020, 10, 30, 22, 59, 59, 0, time.UTC),
			initialMarkPrice:       130000,
			settlementValue:        126000,
			riskAversionParameter:  0.01,
			sigma:                  0.09,
			openingAuctionDuration: "0m20s",
		},
		{
			id:                     "RTJVFCMFZZQQLLYVSXTWEN62P6AH6OCN",
			decimalPlaces:          5,
			baseName:               "ETH",
			quoteName:              "BTC",
			settlementAsset:        "BTC",
			maturity:               time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC),
			initialMarkPrice:       10000,
			settlementValue:        98123,
			riskAversionParameter:  0.001,
			sigma:                  2.0,
			openingAuctionDuration: "0h0m30s",
		},
	}

	filenames := make([]string, len(skels))

	for seq, skel := range skels {
		monYear := skel.maturity.Format("Jan06")
		monYearUpper := strings.ToUpper(monYear)
		auctionDuration, err := time.ParseDuration(skel.openingAuctionDuration)
		if err != nil {
			return nil, err
		}

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
			OpeningAuction: &proto.AuctionDuration{
				Duration: int64(auctionDuration.Seconds()),
				Volume:   0,
			},
			PriceMonitoringSettings: &proto.PriceMonitoringSettings{
				Parameters: &proto.PriceMonitoringParameters{
					Triggers: []*proto.PriceMonitoringTrigger{},
				},
				UpdateFrequency: 60,
			},
			TargetStakeParameters: &proto.TargetStakeParameters{
				TimeWindow:    3600, // seconds = 1h
				ScalingFactor: 10,
			},
		}
		filenames[seq] = fmt.Sprintf("%s%s%s.json", skel.baseName, skel.quoteName, monYearUpper)
		err = createDefaultMarket(&mkt, path.Join(confpath, filenames[seq]), uint64(seq))
		if err != nil {
			return nil, err
		}
	}

	return filenames, nil
}

func Init(ctx context.Context, parser *flags.Parser) error {
	initCmd = InitCmd{
		RootPathFlag: config.NewRootPathFlag(),
	}

	var (
		short = "Initializes a vega node"
		long  = "Generate the minimal configuration required for a vega node to start"
	)
	_, err := parser.AddCommand("init", short, long, &initCmd)
	return err
}

func createDefaultMarket(mkt *proto.Market, path string, seq uint64) error {
	m := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
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
