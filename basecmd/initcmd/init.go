package initcmd

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/basecmd"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/zannen/toml"
)

var (
	Command basecmd.Command

	configPath string
	force      bool
)

func init() {
	Command.Name = "init"
	Command.Short = "Generate an initial vega configuration"

	cmd := flag.NewFlagSet("node", flag.ContinueOnError)
	cmd.StringVar(&configPath, "config-path", fsutil.DefaultVegaDir(), "file path in which the configuration will be located")
	cmd.BoolVar(&force, "f", false, "erase existing configuration at the specified path")

	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "%v\n\n", helpInit())
		cmd.PrintDefaults()
	}

	Command.FlagSet = cmd
	Command.Usage = Command.FlagSet.Usage
	Command.Run = runCommand
}

func helpInit() string {
	helpStr := `
Usage: vega init [options]
`
	return strings.TrimSpace(helpStr)
}

func runCommand(_ *logging.Logger, args []string) int {
	if err := Command.FlagSet.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(Command.FlagSet.Output(), "%v\n", err)
		return 1
	}

	if len(configPath) <= 0 {
		fmt.Fprintln(os.Stderr, "vega: config path cannot be empty")
		return 1
	}

	if err := runInit(configPath, force); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	return 0
}

// RunInit initialises vega config files - config.toml and markets/*.json.
func runInit(rootPath string, force bool) error {
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
		fmt.Printf("removing existing configuration at %v\n", rootPath)
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

	fmt.Printf("configuration generated successfully at %v\n", rootPath)

	return nil
}

func createDefaultMarkets(confpath string) ([]string, error) {
	/*
		Notes on default markets:
		- If decimalPlaces==2, then a currency balance of `1` indicates one Euro cent, not one Euro
		- Maturity dates should be not all the same, for variety.
	*/
	skels := []struct {
		decimalPlaces    uint64
		baseName         string
		settlementAsset  string
		quoteName        string
		maturity         time.Time
		initialMarkPrice uint64
		settlementValue  uint64
	}{
		{
			decimalPlaces:    5,
			baseName:         "ETH",
			quoteName:        "USD",
			settlementAsset:  "VUSD",
			maturity:         time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC),
			initialMarkPrice: 200,
			settlementValue:  200,
		},
		{
			decimalPlaces:    5,
			baseName:         "GBP",
			quoteName:        "USD",
			settlementAsset:  "VUSD",
			maturity:         time.Date(2020, 6, 30, 22, 59, 59, 0, time.UTC),
			initialMarkPrice: 10,
			settlementValue:  10,
		},
		{
			decimalPlaces:    5,
			baseName:         "ETH",
			quoteName:        "BTC",
			settlementAsset:  "BTC",
			maturity:         time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC),
			initialMarkPrice: 5,
			settlementValue:  5,
		},
	}

	filenames := make([]string, len(skels))

	for seq, skel := range skels {
		monYear := skel.maturity.Format("Jan06")
		monYearUpper := strings.ToUpper(monYear)

		mkt := proto.Market{
			Name:          fmt.Sprintf("%s%s/%s", skel.baseName, skel.quoteName, monYearUpper),
			DecimalPlaces: skel.decimalPlaces,
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
		filenames[seq] = fmt.Sprintf("%s%s%s.json", skel.baseName, skel.quoteName, monYearUpper)
		err := createDefaultMarket(&mkt, path.Join(confpath, filenames[seq]), uint64(seq))
		if err != nil {
			return nil, err
		}
	}

	return filenames, nil
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

	if _, err := f.WriteString(buf); err != nil {
		return err
	}

	return nil
}
