package nodewallet

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet/eth"
	"code.vegaprotocol.io/vega/nodewallet/vega"
)

type ChainWallet string

const (
	Vega ChainWallet = "vega"
	Eth  ChainWallet = "eth"
)

var requiredWallets = []ChainWallet{Vega, Eth}

type Wallet interface {
	Chain() string
	Sign([]byte) ([]byte, error)
	PubKeyOrAddress() []byte
}

type Service struct {
	log     *logging.Logger
	cfg     Config
	store   *store
	wallets map[ChainWallet]Wallet
}

func New(log *logging.Logger, cfg Config, passphrase string) (*Service, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	stor, err := loadStore(cfg.StorePath, passphrase)
	if err != nil {
		return nil, fmt.Errorf("unable to load nodewalletsore: %v", err)
	}

	wallets, err := loadWallets(stor)
	if err != nil {
		return nil, fmt.Errorf("error with the wallets stored in the nodewalletstore, %v", err)
	}

	err = ensureRequiredWallets(wallets)
	if err != nil {
		return nil, err
	}

	return &Service{
		log:     log,
		cfg:     cfg,
		store:   stor,
		wallets: wallets,
	}, nil
}

func (s *Service) Get(chain ChainWallet) (Wallet, bool) {
	w, ok := s.wallets[chain]
	return w, ok
}

// this will replace any existing import for a chain
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
	switch ChainWallet(chain) {
	case Vega:
		w, err = vega.New(path, walletPassphrase)
		if err != nil {
			return err
		}
	case Eth:
		w, err = eth.New(path, walletPassphrase)
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
	s.wallets[ChainWallet(chain)] = w
	return saveStore(s.store, s.cfg.StorePath, passphrase)
}

func (s *Service) Dump() error {
	buf, err := json.MarshalIndent(s.store.Wallets, " ", " ")
	if err != nil {
		return fmt.Errorf("unable to indent message: %v", err)
	}

	// print the new keys for user info
	fmt.Printf("%v\n", string(buf))
	return nil
}

func Verify(cfg Config, passphrase string) error {
	store, err := loadStore(cfg.StorePath, passphrase)
	if err != nil {
		return fmt.Errorf("unable to load nodewalletsore: %v", err)
	}

	wallets, err := loadWallets(store)
	if err != nil {
		return fmt.Errorf("error with the wallets stored in the nodewalletstore, %v", err)
	}

	return ensureRequiredWallets(wallets)
}

func IsSupported(chain string) error {
	for _, ch := range requiredWallets {
		if ChainWallet(chain) == ch {
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
		Chain:      string(Eth),
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

func ensureRequiredWallets(wallets map[ChainWallet]Wallet) error {
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
func loadWallets(stor *store) (map[ChainWallet]Wallet, error) {
	wallets := map[ChainWallet]Wallet{}

	for _, w := range stor.Wallets {
		w := w
		if _, ok := wallets[ChainWallet(w.Chain)]; ok {
			return nil, fmt.Errorf("duplicate wallet configuration for chain %v", w)
		}
		switch ChainWallet(w.Chain) {
		case Vega:
			w, err := vega.New(w.Path, w.Passphrase)
			if err != nil {
				return nil, err
			}
			wallets[Vega] = w
		case Eth:
			w, err := eth.New(w.Path, w.Passphrase)
			if err != nil {
				return nil, err
			}
			wallets[Eth] = w
		default:
			return nil, fmt.Errorf("unsupported chain wallet: %v", w.Chain)
		}
	}
	return wallets, nil
}
