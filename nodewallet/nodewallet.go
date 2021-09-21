package nodewallet

import (
	"context"
	"fmt"
	"path/filepath"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet/eth"
	"code.vegaprotocol.io/vega/nodewallet/vega"

	"github.com/pkg/errors"
)

type Blockchain string

const (
	Vega     Blockchain = "vega"
	Ethereum Blockchain = "ethereum"

	namedLogger = "nodewallet"
)

var requiredWallets = []Blockchain{Vega, Ethereum}

type Wallet interface {
	Name() string
	Chain() string
	Sign([]byte) ([]byte, error)
	Algo() string
	Version() uint32
	PubKeyOrAddress() crypto.PublicKeyOrAddress
}

type ETHWallet interface {
	Wallet
	Client() eth.ETHClient
	BridgeAddress() string
	CurrentHeight(context.Context) (uint64, error)
	ConfirmationsRequired() uint32
	SetEthereumConfig(*types.EthereumConfig) error
}

type Service struct {
	log     *logging.Logger
	store   *store
	wallets map[Blockchain]Wallet

	storage *Loader

	ethWalletLoader  *eth.WalletLoader
	vegaWalletLoader *vega.WalletLoader
}

func New(log *logging.Logger, cfg Config, passphrase string, ethClient eth.ETHClient, vegaPaths paths.Paths) (*Service, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	storage, err := InitialiseLoader(vegaPaths, passphrase)
	if err != nil {
		log.Error("couldn't initialise the node wallet store", logging.Error(err))
		return nil, fmt.Errorf("couldn't initialise node wallet store: %v", err)
	}

	store, err := storage.Load(passphrase)
	if err != nil {
		log.Error("couldn't load the node wallet store", logging.Error(err))
		return nil, fmt.Errorf("couldn't load node wallet store: %v", err)
	}

	ethWalletLoader, err := eth.InitialiseWalletLoader(vegaPaths, ethClient)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise Ethereum node wallet loader: %w", err)
	}

	vegaWalletLoader, err := vega.InitialiseWalletLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise Vega node wallet loader: %w", err)
	}

	wallets, err := loadWallets(store, ethWalletLoader, vegaWalletLoader)
	if err != nil {
		log.Error("unable to load a chain wallet", logging.Error(err))
		return nil, fmt.Errorf("unable to load a chain wallet: %v", err)
	}

	return &Service{
		log:              log,
		store:            store,
		wallets:          wallets,
		ethWalletLoader:  ethWalletLoader,
		vegaWalletLoader: vegaWalletLoader,
		storage:          storage,
	}, nil
}

func (s *Service) GetConfigFilePath() string {
	return s.storage.configFilePath
}

func (s *Service) OnEthereumConfigUpdate(ctx context.Context, v interface{}) error {
	ecfg, ok := v.(*types.EthereumConfig)
	if !ok {
		return errors.New("invalid types for Ethereum config")
	}
	w, _ := s.Get(Ethereum)
	return w.(ETHWallet).SetEthereumConfig(ecfg)
}

func (s *Service) Cleanup() error {
	wal := s.wallets[Ethereum]
	return wal.(*eth.Wallet).Cleanup()
}

// ReloadConf is used in order to reload the internal configuration of
// the fee engine
func (s *Service) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}
}

func (s *Service) Get(chain Blockchain) (Wallet, bool) {
	w, ok := s.wallets[chain]
	return w, ok
}

func (s *Service) Generate(chain, passphrase, walletPassphrase string) (map[string]string, error) {
	var (
		err  error
		w    Wallet
		data map[string]string
	)
	switch Blockchain(chain) {
	case Vega:
		w, data, err = s.vegaWalletLoader.Generate(walletPassphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate Vega node wallet: %w", err)
		}
	case Ethereum:
		w, data, err = s.ethWalletLoader.Generate(walletPassphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate Ethereum node wallet: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported chain wallet %v", chain)
	}

	if err = s.saveWallet(chain, passphrase, walletPassphrase, w); err != nil {
		return nil, err
	}

	return data, err
}

func (s *Service) Import(chain, passphrase, walletPassphrase, sourceFilePath string) (map[string]string, error) {
	if !filepath.IsAbs(sourceFilePath) {
		return nil, fmt.Errorf("path to the wallet file need to be absolute")
	}

	var (
		err  error
		w    Wallet
		data map[string]string
	)
	switch Blockchain(chain) {
	case Vega:
		w, data, err = s.vegaWalletLoader.Import(sourceFilePath, walletPassphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't import Vega node wallet: %w", err)
		}
	case Ethereum:
		w, data, err = s.ethWalletLoader.Import(sourceFilePath, walletPassphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't import Ethereum node wallet: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported chain wallet %v", chain)
	}

	if err := s.saveWallet(chain, passphrase, walletPassphrase, w); err != nil {
		return nil, err
	}

	return data, nil
}

func (s *Service) Verify() error {
	for _, v := range requiredWallets {
		_, ok := s.wallets[v]
		if !ok {
			return fmt.Errorf("required wallet for %v chain is missing", v)
		}
	}
	return nil
}

func (s *Service) Show() map[string]WalletConfig {
	configs := map[string]WalletConfig{}
	for _, config := range s.store.Wallets {
		configs[config.Chain] = config
	}
	return configs
}

func (s *Service) saveWallet(chain string, passphrase string, walletPassphrase string, w Wallet) error {
	s.store.AddWallet(WalletConfig{
		Chain:      chain,
		Passphrase: walletPassphrase,
		Name:       w.Name(),
	})
	s.wallets[Blockchain(chain)] = w
	err := s.storage.Save(s.store, passphrase)
	if err != nil {
		return fmt.Errorf("couldn't save node wallets configuration: %w", err)
	}
	return nil
}

// loadWallets takes the wallets configs from the store and try to instantiate
// them to proper blockchains wallets
func loadWallets(store *store, ethWalletLoader *eth.WalletLoader, vegaWalletLoader *vega.WalletLoader) (map[Blockchain]Wallet, error) {
	wallets := map[Blockchain]Wallet{}

	for _, w := range store.Wallets {
		w := w
		if _, ok := wallets[Blockchain(w.Chain)]; ok {
			return nil, fmt.Errorf("duplicate wallet configuration for chain %v", w)
		}

		switch Blockchain(w.Chain) {
		case Vega:
			w, err := vegaWalletLoader.Load(w.Name, w.Passphrase)
			if err != nil {
				return nil, fmt.Errorf("couldn't load Vega node wallet: %w", err)
			}
			wallets[Vega] = w
		case Ethereum:
			w, err := ethWalletLoader.Load(w.Name, w.Passphrase)
			if err != nil {
				return nil, fmt.Errorf("couldn't load Ethereum node wallet: %w", err)
			}
			wallets[Ethereum] = w
		default:
			return nil, fmt.Errorf("unsupported chain wallet: %v", w.Chain)
		}
	}
	return wallets, nil
}
