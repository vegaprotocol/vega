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

package keystore

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/paths"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type WalletLoader struct {
	walletHome string
}

func InitialiseWalletLoader(vegaPaths paths.Paths) (*WalletLoader, error) {
	walletHome, err := vegaPaths.CreateDataDirFor(paths.EthereumNodeWalletsDataHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get the directory path for %s: %w", paths.EthereumNodeWalletsDataHome, err)
	}

	return &WalletLoader{
		walletHome: walletHome,
	}, nil
}

func (l *WalletLoader) Generate(passphrase string) (*Wallet, map[string]string, error) {
	ks := keystore.NewKeyStore(l.walletHome, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(passphrase)
	if err != nil {
		return nil, nil, err
	}

	_, fileName := filepath.Split(acc.URL.Path)

	content, err := vgfs.ReadFile(acc.URL.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read file %s: %w", acc.URL.Path, err)
	}

	w, err := newWallet(l, fileName, passphrase, content)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't create wallet: %w", err)
	}

	data := map[string]string{
		"walletFilePath":  acc.URL.Path,
		"ethereumAddress": w.address.Hex(),
	}

	return w, data, nil
}

func (l *WalletLoader) Load(walletName, passphrase string) (*Wallet, error) {
	data, err := fs.ReadFile(os.DirFS(l.walletHome), walletName)
	if err != nil {
		return nil, fmt.Errorf("couldn't read wallet file: %v", err)
	}

	w, err := newWallet(l, walletName, passphrase, data)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (l *WalletLoader) Import(sourceFilePath, passphrase string) (*Wallet, map[string]string, error) {
	content, err := vgfs.ReadFile(sourceFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read file %s: %w", sourceFilePath, err)
	}

	_, fileName := filepath.Split(sourceFilePath)

	walletFilePath := filepath.Join(l.walletHome, fileName)
	err = vgfs.WriteFile(walletFilePath, content)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't write file %s: %w", walletFilePath, err)
	}

	w, err := newWallet(l, fileName, passphrase, content)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't create wallet: %w", err)
	}

	data := map[string]string{
		"walletFilePath":  walletFilePath,
		"ethereumAddress": w.address.Hex(),
	}

	return w, data, nil
}
