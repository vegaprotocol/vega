package nodewallets

import (
	"encoding/json"
	"errors"
	"fmt"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
)

var (
	errInternalWrongPassphrase = errors.New("couldn't decrypt buffer: cipher: message authentication failed")
	ErrWrongPassphrase         = errors.New("wrong passphrase")
)

const (
	ethereumWalletTypeKeyStore ethereumWalletType = "key-store"
	ethereumWalletTypeClef     ethereumWalletType = "clef"
)

type Registry struct {
	Ethereum *RegisteredEthereumWallet `json:"ethereum,omitempty"`
	Vega     *RegisteredVegaWallet     `json:"vega,omitempty"`
}

type ethereumWalletType string

type ethereumWallet interface {
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
	Type    ethereumWalletType `json:"type"`
	Details ethereumWallet     `json:"details"`
}

func (rw *RegisteredEthereumWallet) UnmarshalJSON(data []byte) error {
	input := struct {
		Type    string          `json:"type"`
		Details json.RawMessage `json:"details"`
	}{}

	if err := json.Unmarshal(data, &input); err != nil {
		return nil
	}

	rw.Type = ethereumWalletType(input.Type)

	switch rw.Type {
	case ethereumWalletTypeKeyStore:
		var keyStore EthereumKeyStoreWallet
		if err := json.Unmarshal(input.Details, &keyStore); err != nil {
			return err
		}

		rw.Details = keyStore
	case ethereumWalletTypeClef:
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

type RegistryLoader struct {
	registryFilePath string
}

func NewRegistryLoader(vegaPaths paths.Paths, passphrase string) (*RegistryLoader, error) {
	registryFilePath, err := vegaPaths.ConfigPathFor(paths.NodeWalletsConfigFile)
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

	return &RegistryLoader{
		registryFilePath: registryFilePath,
	}, nil
}

func (l *RegistryLoader) GetRegistry(passphrase string) (*Registry, error) {
	registry := &Registry{}
	if err := paths.ReadEncryptedFile(l.registryFilePath, passphrase, registry); err != nil {
		if err.Error() == errInternalWrongPassphrase.Error() {
			return nil, ErrWrongPassphrase
		}
		return nil, fmt.Errorf("couldn't read encrypted file %s: %w", l.registryFilePath, err)
	}
	return registry, nil
}

func (l *RegistryLoader) SaveRegistry(registry *Registry, passphrase string) error {
	err := paths.WriteEncryptedFile(l.registryFilePath, passphrase, registry)
	if err != nil {
		return fmt.Errorf("couldn't write encrypted file %s: %w", l.registryFilePath, err)
	}
	return nil
}

func (l *RegistryLoader) RegistryFilePath() string {
	return l.registryFilePath
}
