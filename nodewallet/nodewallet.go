package nodewallet

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	types "code.vegaprotocol.io/protos/vega"
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

	storage *storage

	ethWalletLoader  *eth.WalletLoader
	vegaWalletLoader *vega.WalletLoader
}

func New(log *logging.Logger, cfg Config, passphrase string, ethClient eth.ETHClient, rootPath string) (*Service, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	storage := newStorage(rootPath)
	store, err := storage.Load(passphrase)
	if err != nil {
		log.Error("unable to load the node wallet", logging.Error(err))
		return nil, fmt.Errorf("unable to load store: %v", err)
	}

	ethWalletLoader := eth.NewWalletLoader(storage.WalletDirFor(Ethereum), ethClient)
	err = ethWalletLoader.Initialise()
	if err != nil {
		return nil, err
	}

	vegaWalletLoader := vega.NewWalletLoader(storage.WalletDirFor(Vega))
	err = vegaWalletLoader.Initialise()
	if err != nil {
		return nil, err
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
// the of the fee engine
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

func (s *Service) Generate(chain, passphrase string) error {
	var (
		err error
		w   Wallet
	)
	switch Blockchain(chain) {
	case Vega:
		w, err = s.vegaWalletLoader.Generate(passphrase)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported chain wallet %v", chain)
	}

	s.store.AddWallet(WalletConfig{
		Chain:      chain,
		Passphrase: passphrase,
		Name:       w.Name(),
	})
	s.wallets[Blockchain(chain)] = w
	return s.storage.Save(s.store, passphrase)
}

func (s *Service) Import(chain, passphrase, walletPassphrase, sourceFilePath string) error {
	if !filepath.IsAbs(sourceFilePath) {
		return fmt.Errorf("path to the wallet file need to be absolute")
	}

	var (
		err error
		w   Wallet
	)
	switch Blockchain(chain) {
	case Vega:
		w, err = s.vegaWalletLoader.Import(sourceFilePath, walletPassphrase)
		if err != nil {
			return err
		}
	case Ethereum:
		w, err = s.ethWalletLoader.Import(sourceFilePath, walletPassphrase)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported chain wallet %v", chain)
	}

	s.store.AddWallet(WalletConfig{
		Chain:      chain,
		Passphrase: walletPassphrase,
		Name:       w.Name(),
	})
	s.wallets[Blockchain(chain)] = w
	return s.storage.Save(s.store, passphrase)
}

func (s *Service) Dump() (string, error) {
	buf, err := json.MarshalIndent(s.store.Wallets, " ", " ")
	if err != nil {
		return "", fmt.Errorf("unable to indent message: %v", err)
	}

	return string(buf), nil
}

func (s *Service) Verify() error {
	for _, v := range requiredWallets {
		_, ok := s.wallets[v]
		if !ok {
			return fmt.Errorf("missing required wallet for %v chain", v)
		}
	}
	return nil
}

func Initialise(rootPath, passphrase string) error {
	storage := newStorage(rootPath)
	return storage.Initialise(passphrase)
}

func DevInit(rootPath, passphrase string) error {
	storage := newStorage(rootPath)
	err := storage.Initialise(passphrase)
	if err != nil {
		return err
	}

	cfgs := []WalletConfig{}

	ethWalletName, err := eth.DevInit(storage.WalletDirFor(Ethereum), passphrase)
	if err != nil {
		return err
	}
	cfgs = append(cfgs, WalletConfig{
		Chain:      string(Ethereum),
		Name:       ethWalletName,
		Passphrase: passphrase,
	})

	return storage.Save(&store{Wallets: cfgs}, passphrase)
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
				return nil, err
			}
			wallets[Vega] = w
		case Ethereum:
			w, err := ethWalletLoader.Load(w.Name, w.Passphrase)
			if err != nil {
				return nil, err
			}
			wallets[Ethereum] = w
		default:
			return nil, fmt.Errorf("unsupported chain wallet: %v", w.Chain)
		}
	}
	return wallets, nil
}
