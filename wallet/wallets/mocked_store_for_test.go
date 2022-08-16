package wallets_test

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/wallet/wallet"
	"code.vegaprotocol.io/vega/wallet/wallets"
)

var errWrongPassphrase = errors.New("wrong passphrase")

type mockedStore struct {
	passphrase string
	wallets    map[string]wallet.Wallet
}

func newMockedStore() *mockedStore {
	return &mockedStore{
		passphrase: "",
		wallets:    map[string]wallet.Wallet{},
	}
}

func (m *mockedStore) WalletExists(_ context.Context, name string) (bool, error) {
	_, ok := m.wallets[name]
	return ok, nil
}

func (m *mockedStore) ListWallets(_ context.Context) ([]string, error) {
	ws := make([]string, 0, len(m.wallets))
	for k := range m.wallets {
		ws = append(ws, k)
	}
	return ws, nil
}

func (m *mockedStore) SaveWallet(_ context.Context, w wallet.Wallet, passphrase string) error {
	m.passphrase = passphrase
	m.wallets[w.Name()] = w
	return nil
}

func (m *mockedStore) GetWallet(_ context.Context, name, passphrase string) (wallet.Wallet, error) {
	w, ok := m.wallets[name]
	if !ok {
		return nil, wallets.ErrWalletDoesNotExists
	}
	if passphrase != m.passphrase {
		return nil, errWrongPassphrase
	}
	return w, nil
}

func (m *mockedStore) GetWalletPath(name string) string {
	return fmt.Sprintf("some/path/%v", name)
}

func (m *mockedStore) GetKey(name, pubKey string) wallet.PublicKey {
	w, ok := m.wallets[name]
	if !ok {
		panic(fmt.Sprintf("wallet \"%v\" not found", name))
	}
	for _, key := range w.ListPublicKeys() {
		if key.Key() == pubKey {
			return key
		}
	}
	panic(fmt.Sprintf("key \"%v\" not found", pubKey))
}
