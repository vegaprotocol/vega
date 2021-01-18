package eth

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

// ETHClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_client_mock.go -package mocks code.vegaprotocol.io/vega/nodewallet/eth ETHClient
type ETHClient interface {
	bind.ContractBackend
	ChainID(context.Context) (*big.Int, error)
	NetworkID(context.Context) (*big.Int, error)
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

type Wallet struct {
	cfg        Config
	acc        accounts.Account
	ks         *keystore.KeyStore
	clt        ETHClient
	passphrase string

	pcfg *types.EthereumConfig

	// this is all just to prevent spamming the infura just
	// to get the last height of the blockchain
	mu                  sync.Mutex
	curHeightLastUpdate time.Time
	curHeight           uint64
}

func DevInit(path, passphrase string) (string, error) {
	ks := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(passphrase)
	if err != nil {
		return "", err
	}
	return acc.URL.Path, nil
}

func randomFolder() string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 10)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func New(cfg Config, path, passphrase string, ethclt ETHClient) (*Wallet, error) {
	// NewKeyStore always create a new wallet key store file
	// we create this in tmp as we do not want to impact the original one.
	ks := keystore.NewKeyStore(
		filepath.Join(os.TempDir(), randomFolder()), keystore.StandardScryptN, keystore.StandardScryptP)
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// just trying to call to make sure there's not issue
	_, err = ethclt.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	acc, err := ks.Import(jsonBytes, passphrase, passphrase)
	if err != nil {
		return nil, err
	}

	if err := ks.Unlock(acc, passphrase); err != nil {
		return nil, errors.Wrap(err, "unable to unlock wallet")
	}

	return &Wallet{
		cfg:        cfg,
		acc:        acc,
		ks:         ks,
		clt:        ethclt,
		passphrase: passphrase,
	}, nil
}

func (w *Wallet) SetEthereumConfig(pcfg *types.EthereumConfig) error {
	nid, err := w.clt.NetworkID(context.Background())
	if err != nil {
		return err
	}
	chid, err := w.clt.ChainID(context.Background())
	if err != nil {
		return err
	}
	if nid.String() != pcfg.NetworkId {
		return fmt.Errorf("ethereum network id does not match, expected %v got %v", pcfg.NetworkId, nid)
	}
	if chid.String() != pcfg.ChainId {
		return fmt.Errorf("ethereum chain id does not match, expected %v got %v", pcfg.ChainId, chid)
	}
	w.pcfg = pcfg
	return nil
}

func (w *Wallet) Cleanup() error {
	// just remove the wallet from the tmp file
	return w.ks.Delete(w.acc, w.passphrase)
}

func (w *Wallet) Chain() string {
	return "ethereum"
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	return w.ks.SignHash(w.acc, data)
}

func (w *Wallet) Algo() string {
	return "eth"
}

func (w *Wallet) Version() uint64 {
	return 0
}

func (w *Wallet) PubKeyOrAddress() []byte {
	return w.acc.Address.Bytes()
}

func (w *Wallet) Client() ETHClient {
	return w.clt
}

func (w *Wallet) BridgeAddress() string {
	return w.pcfg.BridgeAddress
}

func (w *Wallet) CurrentHeight(ctx context.Context) (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// if last update of the heigh was more that 15 seconds
	// ago, we try to update, we assume an eth block takes
	// ~15 seconds
	now := time.Now()
	if w.curHeightLastUpdate.Add(15).Before(now) {
		// getthe last block header
		h, err := w.clt.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return w.curHeight, err
		}
		w.curHeightLastUpdate = now
		w.curHeight = h.Number.Uint64()
	}

	return w.curHeight, nil
}

func (w *Wallet) ConfirmationsRequired() uint32 {
	return w.pcfg.Confirmations
}
