package eth

import (
	"context"
	"fmt"
	"sync"
	"time"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type Wallet struct {
	name       string
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
	address             crypto.PublicKeyOrAddress
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

func (w *Wallet) Name() string {
	return w.name
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

func (w *Wallet) Version() uint32 {
	return 0
}

func (w *Wallet) PubKeyOrAddress() crypto.PublicKeyOrAddress {
	return w.address
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
		// get the last block header
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
