// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

func (m *mockedStore) UnlockWallet(_ context.Context, name, passphrase string) error {
	_, ok := m.wallets[name]
	if !ok {
		return wallets.ErrWalletDoesNotExists
	}
	if passphrase != m.passphrase {
		return errWrongPassphrase
	}
	return nil
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

func (m *mockedStore) CreateWallet(_ context.Context, w wallet.Wallet, passphrase string) error {
	m.passphrase = passphrase
	m.wallets[w.Name()] = w
	return nil
}

func (m *mockedStore) UpdateWallet(_ context.Context, w wallet.Wallet) error {
	m.wallets[w.Name()] = w
	return nil
}

func (m *mockedStore) GetWallet(_ context.Context, name string) (wallet.Wallet, error) {
	w, ok := m.wallets[name]
	if !ok {
		return nil, wallets.ErrWalletDoesNotExists
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
