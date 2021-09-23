package eth

import (
	"fmt"

	"code.vegaprotocol.io/vega/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type ClefWallet struct {
	name   string
	acc    accounts.Account
	client *rpc.Client
}

func NewClefWallet(endpoint string) (*ClefWallet, error) {
	client, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}

	return &ClefWallet{
		client: client,
		name:   "clef",
	}, nil
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

func (w *ClefWallet) NewAccount() ([]byte, error) {
	var v string
	if err := w.client.Call(&v, "new_account"); err != nil {
		return "", err
	}
}

func (w *ClefWallet) Sign(account accounts.Account, data []byte) ([]byte, error) {
	var res hexutil.Bytes
	var signAddress = common.NewMixedcaseAddress(account.Address)
	if err := w.client.Call(&res, "account_signData",
		accounts.MimetypeTypedData,
		&signAddress, // Need to use the pointer here, because of how MarshalJSON is defined
		hexutil.Encode(data)); err != nil {
		return nil, err
	}
	// If V is on 27/28-form, convert to 0/1 for Clique
	if mimeType == accounts.MimetypeClique && (res[64] == 27 || res[64] == 28) {
		res[64] -= 27 // Transform V from 27/28 to 0/1 for Clique use
	}
	return res, nil
}

func (w *ClefWallet) Algo() string {
	return "eth"
}

func (w *ClefWallet) Version() string {
	v, _ := w.version()

	return v
}

func (w *ClefWallet) Address() crypto.PublicKeyOrAddress {
	return crypto.NewPublicKeyOrAddress("", w.acc.Address.Bytes())
}

func (w *ClefWallet) version() (string, error) {
	var v string
	if err := w.client.Call(&v, "account_version"); err != nil {
		return "", err
	}
	return v, nil
}
