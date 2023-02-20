package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/cmd/vegaone/config"
	"code.vegaprotocol.io/vega/cmd/vegaone/flags"
	cconfig "code.vegaprotocol.io/vega/core/config"
	encoding "code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/core/nodewallets/registry"
	dnconfig "code.vegaprotocol.io/vega/datanode/config"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
	lencoding "code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	tmcfg "github.com/tendermint/tendermint/config"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
)

type initFlags struct {
	globalFlags

	Force         bool
	Passphrase    string
	Mode          string
	WithDatanode  bool
	ChainID       string
	RetentionMode string
	EmbedPostgres bool
}

func (g *initFlags) Register(fset *flag.FlagSet) {
	g.globalFlags.Register(fset)
	fset.BoolVar(&g.EmbedPostgres, "embed-postgres", false, "use the embeded postgres instance")
	fset.BoolVar(&g.Force, "force", false, "should the command erase existing configuration")
	fset.StringVar(&g.Passphrase, "nodewallet-passphrase-file", "", "an optional file containing the passphrase for the node wallet")
	fset.StringVar(&g.ChainID, "chainid", "", "the id of the chain this node with be joining (required only when setting up the datanode)")
	fset.StringVar(&g.Mode, "mode", "validator", "the mode of the vega node [validator|full]")
	fset.StringVar(&g.RetentionMode, "retention-mode", "standard", "the retention mode of the data node state [standard|archive|lite]")
	fset.BoolVar(&g.WithDatanode, "with-datanode", false, "initialise the datanode as well as the core node")
}

type initCommand struct {
	flags initFlags
	fset  *flag.FlagSet
}

func newInit() (i *initCommand) {
	defer func() { i.flags.Register(i.fset) }()
	return &initCommand{
		flags: initFlags{},
		fset:  flag.NewFlagSet("init", flag.ExitOnError),
	}
}

func (i *initCommand) Parse(args []string) error {
	return i.fset.Parse(args)
}

func (i *initCommand) Execute() error {
	logger := configureLogger()
	defer logger.AtExit()

	home := os.ExpandEnv(i.flags.Home)
	tendermintHome := filepath.Join(home, "tendermint")
	vegaPaths := paths.New(home)

	// first check datanode specific error handling
	if i.flags.WithDatanode && len(i.flags.ChainID) <= 0 {
		return errors.New("chain-id is required when setting up a datanode")
	}

	// always init vega
	if err := i.initVega(logger, vegaPaths, i.flags.WithDatanode); err != nil {
		return fmt.Errorf("couldn't initialise vega %w", err)
	}

	// always init tendermint
	if err := i.initTendermint(logger, tendermintHome); err != nil {
		return fmt.Errorf("couldn't initialise tendermint %w", err)
	}

	// init datanode if users want's it
	if i.flags.WithDatanode {
		if err := i.initDatanode(logger, vegaPaths, i.flags.EmbedPostgres); err != nil {
			return fmt.Errorf("couldn't initialise data node %w", err)
		}
	}

	// then final the vegaone config
	c := config.Config{
		WithDatanode: i.flags.WithDatanode,
	}

	path, err := config.Save(home, &c)
	if err != nil {
		return fmt.Errorf("couldn't save vegaone configuration file: %w", err)
	}

	logger.Info("vegone configuration generated successfully", logging.String("path", path))

	return nil
}

func (i *initCommand) initDatanode(
	logger *logging.Logger, vegaPaths paths.Paths, embedPostgres bool,
) error {
	opts := i.flags

	cfgLoader, err := dnconfig.InitialiseLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise configuration loader: %w", err)
	}

	configExists, err := cfgLoader.ConfigExists()
	if err != nil {
		return fmt.Errorf("couldn't verify configuration presence: %w", err)
	}

	if configExists && !opts.Force {
		return fmt.Errorf("configuration already exists at `%s` please remove it first or re-run using -f", cfgLoader.ConfigFilePath())
	}

	if configExists && opts.Force {
		cfgLoader.Remove()
	}

	cfg := dnconfig.NewDefaultConfig()

	cfg.SQLStore.UseEmbedded = true

	cfg.Broker.SocketConfig.IP = "vega_datanode"
	cfg.Broker.SocketConfig.TransportType = "inproc"
	cfg.Broker.SocketConfig.Port = 1789

	mode, err := encoding.DataNodeRetentionModeFromString(opts.RetentionMode)
	if err != nil {
		return err
	}

	switch mode {
	case encoding.DataNodeRetentionModeArchive:
		cfg.NetworkHistory.Store.HistoryRetentionBlockSpan = math.MaxInt64
		cfg.SQLStore.RetentionPeriod = sqlstore.RetentionPeriodArchive
	case encoding.DataNodeRetentionModeLite:
		cfg.SQLStore.RetentionPeriod = sqlstore.RetentionPeriodLite
	}

	cfg.ChainID = opts.ChainID

	if err := cfgLoader.Save(&cfg); err != nil {
		return fmt.Errorf("couldn't save configuration file: %w", err)
	}

	logger.Info("datanode configuration generated successfully", logging.String("path", cfgLoader.ConfigFilePath()))

	return nil

}

func (i *initCommand) initVega(
	logger *logging.Logger,
	vegaPaths paths.Paths,
	withDatanode bool,
) error {
	opts := i.flags
	if len(opts.Mode) <= 0 {
		return errors.New("missing node mode")
	}

	mode, err := encoding.NodeModeFromString(opts.Mode)
	if err != nil {
		return err
	}

	// a nodewallet will be required only for a validator node
	if mode == encoding.NodeModeValidator {
		pass, err := flags.Passphrase(i.flags.Passphrase).Get("nodewallet passphrase", true)
		if err != nil {
			return err
		}

		_, err = registry.NewLoader(vegaPaths, pass)
		if err != nil {
			return err
		}
	}

	cfgLoader, err := cconfig.InitialiseLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise configuration loader: %w", err)
	}

	configExists, err := cfgLoader.ConfigExists()
	if err != nil {
		return fmt.Errorf("couldn't verify configuration presence: %w", err)
	}

	if configExists && !opts.Force {
		return fmt.Errorf("configuration already exists at `%s` please remove it first or re-run using -f", cfgLoader.ConfigFilePath())
	}

	if configExists && opts.Force {
		logger.Info("removing existing configuration", logging.String("path", cfgLoader.ConfigFilePath()))
		cfgLoader.Remove()
	}

	cfg := cconfig.NewDefaultConfig()
	cfg.NodeMode = mode
	cfg.Broker.Socket.Address = "vega_datanode"
	cfg.Broker.Socket.Transport = "inproc"
	cfg.Broker.Socket.Port = 1789
	cfg.Broker.Socket.Enabled = lencoding.Bool(withDatanode)
	cfg.SetDefaultMaxMemoryPercent()

	if err := cfgLoader.Save(&cfg); err != nil {
		return fmt.Errorf("couldn't save configuration file: %w", err)
	}

	logger.Info("core configuration generated successfully",
		logging.String("path", cfgLoader.ConfigFilePath()))

	return nil
}

func (i *initCommand) initTendermint(logger *logging.Logger, tendermintHome string) error {
	config := tmcfg.DefaultConfig()
	config.SetRoot(tendermintHome)
	tmcfg.EnsureRoot(config.RootDir)

	config.Consensus.TimeoutCommit = 0
	config.Consensus.CreateEmptyBlocks = true
	// enforce using priority mempool
	config.Mempool.Version = "v1"
	// enforce compatibility
	config.P2P.MaxPacketMsgPayloadSize = 16384

	// private validator
	privValKeyFile := config.PrivValidatorKeyFile()
	privValStateFile := config.PrivValidatorStateFile()
	var pv *privval.FilePV
	if tmos.FileExists(privValKeyFile) {
		pv = privval.LoadFilePV(privValKeyFile, privValStateFile)
		logger.Info("found private validator",
			logging.String("keyFile", privValKeyFile),
			logging.String("stateFile", privValStateFile),
		)
	} else {
		pv = privval.GenFilePV(privValKeyFile, privValStateFile)
		pv.Save()
		logger.Info("generated private validator",
			logging.String("keyFile", privValKeyFile),
			logging.String("stateFile", privValStateFile),
		)
	}

	nodeKeyFile := config.NodeKeyFile()
	if tmos.FileExists(nodeKeyFile) {
		logger.Info("found node key", logging.String("path", nodeKeyFile))
	} else {
		if _, err := p2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			return err
		}
		logger.Info("generated node key", logging.String("path", nodeKeyFile))
	}

	// genesis file
	genFile := config.GenesisFile()
	if tmos.FileExists(genFile) {
		logger.Info("found genesis file", logging.String("path", genFile))
	} else {
		genDoc := types.GenesisDoc{
			ChainID:         fmt.Sprintf("test-chain-%v", tmrand.Str(6)),
			GenesisTime:     time.Now().Round(0).UTC(),
			ConsensusParams: types.DefaultConsensusParams(),
		}
		pubKey, err := pv.GetPubKey()
		if err != nil {
			return fmt.Errorf("can't get pubkey: %w", err)
		}
		genDoc.Validators = []types.GenesisValidator{{
			Address: pubKey.Address(),
			PubKey:  pubKey,
			Power:   10,
		}}

		if err := genDoc.SaveAs(genFile); err != nil {
			return err
		}
		logger.Info("generated genesis file", logging.String("path", genFile))
	}

	return nil
}

func configureLogger() *logging.Logger {
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,
			TimeKey:     "time",
			EncodeTime:  zapcore.RFC3339TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}
	return logging.NewLoggerFromZapConfig(cfg)
}
