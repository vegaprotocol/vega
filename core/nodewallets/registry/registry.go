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

package registry

import (
	"encoding/json"
	"errors"
	"fmt"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/paths"
)

var (
	errInternalWrongPassphrase = errors.New("couldn't decrypt buffer: cipher: message authentication failed")
	ErrWrongPassphrase         = errors.New("wrong passphrase")
)

const (
	EthereumWalletTypeKeyStore ethereumWalletType = "key-store"
	EthereumWalletTypeClef     ethereumWalletType = "clef"
)

type Registry struct {
	Tendermint *RegisteredTendermintPubkey `json:"tendermint,omitempty"`
	Ethereum   *RegisteredEthereumWallet   `json:"ethereum,omitempty"`
	Vega       *RegisteredVegaWallet       `json:"vega,omitempty"`
}

type ethereumWalletType string

type EthereumWalletDetails interface {
	ETHWallet()
}

type EthereumKeyStoreWallet struct {
	Name       string `json:"name"`
	Passphrase string `json:"passphrase"`
}

func (e EthereumKeyStoreWallet) ETHWallet() {}

type EthereumClefWallet struct {
	Name           string `json:"name"`
	AccountAddress string `json:"account-address"`
	ClefAddress    string `json:"clef-address"`
}

func (e EthereumClefWallet) ETHWallet() {}

type RegisteredEthereumWallet struct {
	Type    ethereumWalletType    `json:"type"`
	Details EthereumWalletDetails `json:"details"`
}

func (rw *RegisteredEthereumWallet) UnmarshalJSON(data []byte) error {
	input := struct {
		Type    string          `json:"type"`
		Details json.RawMessage `json:"details"`
	}{}

	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	rw.Type = ethereumWalletType(input.Type)

	switch rw.Type {
	case EthereumWalletTypeKeyStore:
		var keyStore EthereumKeyStoreWallet
		if err := json.Unmarshal(input.Details, &keyStore); err != nil {
			return err
		}

		rw.Details = keyStore
	case EthereumWalletTypeClef:
		var clef EthereumClefWallet
		if err := json.Unmarshal(input.Details, &clef); err != nil {
			return err
		}

		rw.Details = clef
	default:
		return fmt.Errorf("unknown Ethereum wallet type %s", rw.Type)
	}

	return nil
}

type RegisteredVegaWallet struct {
	Name       string `json:"name"`
	Passphrase string `json:"passphrase"`
}

type RegisteredTendermintPubkey struct {
	Pubkey string `json:"pubkey"`
}

type Loader struct {
	registryFilePath string
}

func NewLoader(vegaPaths paths.Paths, passphrase string) (*Loader, error) {
	registryFilePath, err := vegaPaths.CreateConfigPathFor(paths.NodeWalletsConfigFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get config path for %s: %w", paths.NodeWalletsConfigFile, err)
	}

	exists, err := vgfs.FileExists(registryFilePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't verify the presence of %s: %w", paths.NodeWalletsConfigFile, err)
	}
	if !exists {
		err := paths.WriteEncryptedFile(registryFilePath, passphrase, &Registry{})
		if err != nil {
			return nil, fmt.Errorf("couldn't write default file %s: %w", registryFilePath, err)
		}
	}

	return &Loader{
		registryFilePath: registryFilePath,
	}, nil
}

func (l *Loader) Get(passphrase string) (*Registry, error) {
	registry := &Registry{}
	if err := paths.ReadEncryptedFile(l.registryFilePath, passphrase, registry); err != nil {
		if err.Error() == errInternalWrongPassphrase.Error() {
			return nil, ErrWrongPassphrase
		}
		return nil, fmt.Errorf("couldn't read encrypted file %s: %w", l.registryFilePath, err)
	}
	return registry, nil
}

func (l *Loader) Save(registry *Registry, passphrase string) error {
	err := paths.WriteEncryptedFile(l.registryFilePath, passphrase, registry)
	if err != nil {
		return fmt.Errorf("couldn't write encrypted file %s: %w", l.registryFilePath, err)
	}
	return nil
}

func (l *Loader) RegistryFilePath() string {
	return l.registryFilePath
}
