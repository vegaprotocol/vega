package clef

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

const requestTimeout = time.Second * 5

// TODO make decision about this
type client interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

type wallet struct {
	client   *rpc.Client
	endpoint string
	name     string
	account  *accounts.Account
}

func newAccount(accountAddr ethcommon.Address, endpoint string) *accounts.Account {
	return &accounts.Account{
		URL: accounts.URL{
			Scheme: "clef",
			Path:   endpoint,
		},
		Address: accountAddr,
	}
}

func NewWallet(endpoint string, accountAddr ethcommon.Address) (*wallet, error) {
	client, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to dial Clef daemon: %w", err)
	}

	w := &wallet{
		name:     fmt.Sprintf("clef-%s", endpoint),
		client:   client,
		endpoint: endpoint,
	}

	if !w.contains(accountAddr) {
		return nil, fmt.Errorf("account with address %q not found", accountAddr)
	}

	w.account = newAccount(accountAddr, w.endpoint)

	return w, nil
}

func GenerateNewWallet(endpoint string) (*wallet, error) {
	client, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to dial Clef daemon: %w", err)
	}

	w := &wallet{
		name:     fmt.Sprintf("clef-%s", endpoint),
		client:   client,
		endpoint: endpoint,
	}

	acc, err := w.generateAccount()
	if err != nil {
		return nil, err
	}

	w.account = acc

	return w, nil
}

func (w *wallet) generateAccount() (*accounts.Account, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	var res string
	if err := w.client.CallContext(ctx, &res, "account_new"); err != nil {
		return nil, err
	}

	return newAccount(ethcommon.HexToAddress(res), w.endpoint), nil
}

func (w *wallet) contains(testAddr ethcommon.Address) bool {
	addresses, err := w.listAccounts()
	if err != nil {
		// TODO log the error here
		return false
	}

	for _, addr := range addresses {
		if testAddr == addr {
			return true
		}
	}

	return false
}

func (w *wallet) listAccounts() ([]common.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	var res []common.Address
	if err := w.client.CallContext(ctx, &res, "account_list"); err != nil {
		return nil, err
	}
	return res, nil
}

// Cleanup is noop
func (w *wallet) Cleanup() error {
	return nil
}

func (w *wallet) Name() string {
	return w.name
}

func (w *wallet) Chain() string {
	return "ethereum"
}

func (w *wallet) Sign(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	var res hexutil.Bytes
	signAddress := common.NewMixedcaseAddress(w.account.Address)

	if err := w.client.CallContext(
		ctx,
		&res,
		"account_signData",
		accounts.MimetypeTypedData,
		&signAddress, // Need to use the pointer here, because of how MarshalJSON is defined
		hexutil.Encode(data),
	); err != nil {
		return nil, err
	}

	return res, nil
}

func (w *wallet) Algo() string {
	return "eth"
}

func (w *wallet) Version() string {
	var v string
	if err := w.client.Call(&v, "account_version"); err != nil {
		return ""
	}

	return v
}

func (w *wallet) PubKeyOrAddress() crypto.PublicKeyOrAddress {
	return crypto.NewPublicKeyOrAddress(w.account.Address.Hex(), w.account.Address.Bytes())
}
