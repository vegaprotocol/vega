package eth

import (
	"context"
	"fmt"
	"sync"
	"time"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/rpc"
)

type ClefWallet struct {
	name   string
	acc    accounts.Account
	client *rpc.Client

	ethClient  ETHClient
	passphrase string

	pcfg *types.EthereumConfig

	// this is all just to prevent spamming the infura just
	// to get the last height of the blockchain
	mu                  sync.Mutex
	curHeightLastUpdate time.Time
	curHeight           uint64
	address             crypto.PublicKeyOrAddress
}

func NewClefWallet(endpoint string, ethClient ETHClient) (*ClefWallet, error) {
	client, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}

	return &ClefWallet{
		client: client,
		name:   "clef",
	}, nil
}

func (w *ClefWallet) SetEthereumConfig(pcfg *types.EthereumConfig) error {
	nid, err := w.ethClient.NetworkID(context.Background())
	if err != nil {
		return err
	}
	chid, err := w.ethClient.ChainID(context.Background())
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

func (w *ClefWallet) Cleanup() error {
	return fmt.Errorf("operation not supported on external signers")
}

func (w *ClefWallet) Name() string {
	return w.name
}

func (w *ClefWallet) Chain() string {
	return "ethereum"
}

func (w *ClefWallet) Sign(data []byte) ([]byte, error) {
	return w.ks.SignHash(w.acc, data)
}

func (w *ClefWallet) Algo() string {
	return "eth"
}

func (w *ClefWallet) Version() uint32 {
	w.client.Call(result interface{}, method string, args ...interface{})
	return 0
}

func (w *ClefWallet) PubKeyOrAddress() crypto.PublicKeyOrAddress {
	return w.address
}

func (w *ClefWallet) Client() ETHClient {
	return w.ethClient
}

func (w *ClefWallet) BridgeAddress() string {
	return w.pcfg.BridgeAddress
}

func (w *ClefWallet) CurrentHeight(ctx context.Context) (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// if last update of the heigh was more that 15 seconds
	// ago, we try to update, we assume an eth block takes
	// ~15 seconds
	now := time.Now()
	if w.curHeightLastUpdate.Add(15).Before(now) {
		// get the last block header
		h, err := w.ethClient.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return w.curHeight, err
		}
		w.curHeightLastUpdate = now
		w.curHeight = h.Number.Uint64()
	}

	return w.curHeight, nil
}

func (w *ClefWallet) ConfirmationsRequired() uint32 {
	return w.pcfg.Confirmations
}

func (w *ClefWallet) version() (string, error) {
	var v string
	if err := api.client.Call(&v, "account_version"); err != nil {
		return "", err
	}
	return v, nil
}