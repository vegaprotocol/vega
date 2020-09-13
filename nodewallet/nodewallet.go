package nodewallet

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"

	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet/eth"
	"code.vegaprotocol.io/vega/nodewallet/vega"
	"github.com/pkg/errors"
)

type Blockchain string

const (
	Vega     Blockchain = "vega"
	Ethereum Blockchain = "ethereum"
)

var requiredWallets = []Blockchain{Vega, Ethereum}

type Wallet interface {
	Chain() string
	Sign([]byte) ([]byte, error)
	Algo() string
	Version() uint64
	PubKeyOrAddress() []byte
}

type ETHWallet interface {
	Wallet
	Client() eth.ETHClient
	BridgeAddress() string
}

type Service struct {
	log     *logging.Logger
	cfg     Config
	store   *store
	wallets map[Blockchain]Wallet

	ethclt eth.ETHClient
}

func New(log *logging.Logger, cfg Config, passphrase string, ethclt eth.ETHClient) (*Service, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	stor, err := loadStore(cfg.StorePath, passphrase)
	if err != nil {
		log.Error("unable to load the node wallet", logging.Error(err))
		return nil, fmt.Errorf("unable to load nodewalletsore: %v", err)
	}

	wallets, err := loadWallets(cfg, stor, ethclt)
	if err != nil {
		log.Error("unable to load a chain wallet", logging.Error(err))
		return nil, fmt.Errorf("error with the wallets stored in the nodewalletstore, %v", err)
	}

	return &Service{
		log:     log,
		cfg:     cfg,
		store:   stor,
		wallets: wallets,
		ethclt:  ethclt,
	}, nil
}

// UponGenesis load the genesis configuration into the governance engine
func (s *Service) UponGenesis(rawState []byte) error {
	s.log.Debug("loading genesis configuration")
	state, err := LoadGenesisState(rawState)
	if err != nil {
		s.log.Error("unable to load genesis state",
			logging.Error(err))
		return err
	}

	// ensure the state is loaded properly
	if state == nil {
		return errors.New("missing genesis state")
	}
	if len(state.ETH.ChainID) <= 0 {
		return errors.New("missing chain ID")
	}
	if len(state.ETH.ERC20BridgeAddress) <= 0 {
		return errors.New("missing erc20 bridge address")
	}

	// first validate chain ID
	chainID, err := s.ethclt.ChainID(context.Background())
	if err != nil {
		return err
	}
	stateChainID, ok := big.NewInt(0).SetString(state.ETH.ChainID, 10)
	if !ok {
		return fmt.Errorf("unable to load eth chain ID: %v", state.ETH.ChainID)
	}
	if chainID.Cmp(stateChainID) != 0 {
		return fmt.Errorf("invalid eth chain ID, expected %v got %v",
			chainID.String(), stateChainID.String())
	}

	// then set the ETH wallet with the BridgeAddress
	ethwal := s.wallets[Ethereum].(*eth.Wallet)
	ethwal.SetERC20BridgeAddress(state.ETH.ERC20BridgeAddress)

	return nil
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

	s.cfg = cfg
}

func (s *Service) EnsureRequireWallets() error {
	return ensureRequiredWallets(s.wallets)
}

func (s *Service) Get(chain Blockchain) (Wallet, bool) {
	w, ok := s.wallets[chain]
	return w, ok
}

// Import replaces any existing import for a chain
func (s *Service) Import(chain, passphrase, walletPassphrase, path string) error {
	// check if the filepath is absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path to the wallet file need to be absolute")
	}

	// try to load the wallet
	// if an error occur return, else we can proceed to save the wallet after.
	var (
		err error
		w   Wallet
	)
	switch Blockchain(chain) {
	case Vega:
		w, err = vega.New(path, walletPassphrase)
		if err != nil {
			return err
		}
	case Ethereum:
		w, err = eth.New(s.cfg.ETH, path, walletPassphrase, s.ethclt)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported chain wallet %v", chain)
	}

	// ok at this point we know the wallet is OK
	// let's add it to the store, and save it again
	s.store.AddWallet(WalletConfig{
		Chain:      chain,
		Passphrase: walletPassphrase,
		Path:       path,
	})
	s.wallets[Blockchain(chain)] = w
	return saveStore(s.store, s.cfg.StorePath, passphrase)
}

func (s *Service) Dump() (string, error) {
	buf, err := json.MarshalIndent(s.store.Wallets, " ", " ")
	if err != nil {
		return "", fmt.Errorf("unable to indent message: %v", err)
	}

	// print the new keys for user info
	return string(buf), nil
}

func Verify(cfg Config, passphrase string, ethclt eth.ETHClient) error {
	store, err := loadStore(cfg.StorePath, passphrase)
	if err != nil {
		return fmt.Errorf("unable to load nodewalletsore: %v", err)
	}

	wallets, err := loadWallets(cfg, store, ethclt)
	if err != nil {
		return fmt.Errorf("error with the wallets stored in the nodewalletstore, %v", err)
	}

	return ensureRequiredWallets(wallets)
}

func IsSupported(chain string) error {
	for _, ch := range requiredWallets {
		if Blockchain(chain) == ch {
			return nil
		}
	}
	return fmt.Errorf("unsupported chain wallet %v", chain)
}

func Init(path, passphrase string) error {
	return saveStore(&store{Wallets: []WalletConfig{}}, path, passphrase)
}

func DevInit(path, devKeyPath, passphrase string) error {
	if ok, _ := fsutil.PathExists(path); ok {
		return fmt.Errorf("dev wallet folder already exists %v", path)
	}

	cfgs := []WalletConfig{}

	// generate eth keys
	ethWalletPath, err := eth.DevInit(devKeyPath, passphrase)
	if err != nil {
		return err
	}
	cfgs = append(cfgs, WalletConfig{
		Chain:      string(Ethereum),
		Path:       ethWalletPath,
		Passphrase: passphrase,
	})
	// generate the vega keys
	vegaWalletPath, err := vega.DevInit(devKeyPath, passphrase)
	if err != nil {
		return err
	}
	cfgs = append(cfgs, WalletConfig{
		Chain:      string(Vega),
		Path:       vegaWalletPath,
		Passphrase: passphrase,
	})

	return saveStore(&store{Wallets: cfgs}, path, passphrase)
}

func ensureRequiredWallets(wallets map[Blockchain]Wallet) error {
	for _, v := range requiredWallets {
		_, ok := wallets[v]
		if !ok {
			return fmt.Errorf("missing required wallet for %v chain", v)
		}
	}
	return nil
}

// takes the wallets configs from the store and try to instanciate them
// to proper blockchains wallets
func loadWallets(cfg Config, stor *store, ethclt eth.ETHClient) (map[Blockchain]Wallet, error) {
	wallets := map[Blockchain]Wallet{}

	for _, w := range stor.Wallets {
		w := w
		if _, ok := wallets[Blockchain(w.Chain)]; ok {
			return nil, fmt.Errorf("duplicate wallet configuration for chain %v", w)
		}

		switch Blockchain(w.Chain) {
		case Vega:
			w, err := vega.New(w.Path, w.Passphrase)
			if err != nil {
				return nil, err
			}
			wallets[Vega] = w
		case Ethereum:
			w, err := eth.New(cfg.ETH, w.Path, w.Passphrase, ethclt)
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
