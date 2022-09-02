// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package clef

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/nodewallets/registry"
	"code.vegaprotocol.io/vega/libs/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	requestTimeout          = time.Second * 10
	signDataTextRawMimeType = "text/raw"
	ClefAlgoType            = "clef"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/rpc_client_mock.go -package mocks code.vegaprotocol.io/vega/core/nodewallets/eth/clef Client
type Client interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	Close()
}

type Wallet struct {
	client   Client
	endpoint string
	name     string
	account  *accounts.Account
	mut      sync.Mutex
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

func NewWallet(client Client, endpoint string, accountAddr ethcommon.Address) (*Wallet, error) {
	w := &Wallet{
		name:     fmt.Sprintf("clef-%s", endpoint),
		client:   client,
		endpoint: endpoint,
	}

	if err := w.contains(accountAddr); err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	w.account = newAccount(accountAddr, w.endpoint)

	return w, nil
}

// GenerateNewWallet new wallet will create new account in Clef and returns wallet.
// Caveat: generating new wallet in Clef has to be manually approved and only key store backend is supported.
func GenerateNewWallet(client Client, endpoint string) (*Wallet, error) {
	w := &Wallet{
		name:     fmt.Sprintf("clef-%s", endpoint),
		client:   client,
		endpoint: endpoint,
	}

	acc, err := w.generateAccount()
	if err != nil {
		return nil, fmt.Errorf("failed to generate account: %w", err)
	}

	w.account = acc

	return w, nil
}

func (w *Wallet) generateAccount() (*accounts.Account, error) {
	// increase timeout here as generating new account has to be manually approved in Clef
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout*20)
	defer cancel()

	var res string
	if err := w.client.CallContext(ctx, &res, "account_new"); err != nil {
		return nil, fmt.Errorf("failed to call client: %w", err)
	}

	return newAccount(ethcommon.HexToAddress(res), w.endpoint), nil
}

// contains returns nil if account is found, otherwise returns an error.
func (w *Wallet) contains(testAddr ethcommon.Address) error {
	addresses, err := w.listAccounts()
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	for _, addr := range addresses {
		if testAddr == addr {
			return nil
		}
	}

	return fmt.Errorf("wallet does not contain account %q", testAddr)
}

func (w *Wallet) listAccounts() ([]ethcommon.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	var res []ethcommon.Address
	if err := w.client.CallContext(ctx, &res, "account_list"); err != nil {
		return nil, fmt.Errorf("failed to call client: %w", err)
	}
	return res, nil
}

func (w *Wallet) Cleanup() error {
	w.client.Close()
	return nil
}

func (w *Wallet) Name() string {
	return w.name
}

func (w *Wallet) Chain() string {
	return "ethereum"
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	var res hexutil.Bytes
	signAddress := ethcommon.NewMixedcaseAddress(w.account.Address)

	if err := w.client.CallContext(
		ctx,
		&res,
		"account_signData",
		signDataTextRawMimeType,
		&signAddress, // Need to use the pointer here, because of how MarshalJSON is defined
		hexutil.Encode(data),
	); err != nil {
		return nil, fmt.Errorf("failed to call client: %w", err)
	}

	return res, nil
}

func (w *Wallet) Algo() string {
	return ClefAlgoType
}

func (w *Wallet) Version() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	var v string
	if err := w.client.CallContext(ctx, &v, "account_version"); err != nil {
		return "", fmt.Errorf("failed to call client: %w", err)
	}

	return v, nil
}

func (w *Wallet) PubKey() crypto.PublicKey {
	return crypto.NewPublicKey(w.account.Address.Hex(), w.account.Address.Bytes())
}

func (w *Wallet) Reload(details registry.EthereumWalletDetails) error {
	d, ok := details.(registry.EthereumClefWallet)
	if !ok {
		// this would mean an implementation error
		panic(fmt.Errorf("failed to get EthereumClefWallet"))
	}

	accountAddr := ethcommon.HexToAddress(d.AccountAddress)
	if err := w.contains(accountAddr); err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	w.mut.Lock()
	defer w.mut.Unlock()

	w.account = newAccount(accountAddr, w.endpoint)

	return nil
}
