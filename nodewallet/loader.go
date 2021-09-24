package nodewallet

import (
	"errors"
	"fmt"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
)

var (
	errInternalWrongPassphrase = errors.New("couldn't decrypt buffer: cipher: message authentication failed")
	ErrWrongPassphrase         = errors.New("wrong passphrase")
)

type WalletConfig struct {
	Chain      string `json:"chain"`
	Name       string `json:"name"`
	Passphrase string `json:"passphrase"`
}

type store struct {
	Wallets []WalletConfig
}

func (s *store) AddWallet(w WalletConfig) {
	for i, v := range s.Wallets {
		if v.Chain == w.Chain {
			s.Wallets[i] = w
			return
		}
	}
	s.Wallets = append(s.Wallets, w)
}

type Loader struct {
	configFilePath string
	walletsPath    string
}

func InitialiseLoader(vegaPaths paths.Paths, passphrase string) (*Loader, error) {
	configFilePath, err := vegaPaths.ConfigPathFor(paths.NodeWalletsConfigFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get config path for %s: %w", paths.NodeWalletsConfigFile, err)
	}
	exists, err := vgfs.FileExists(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't verify the presence of %s: %w", paths.NodeWalletsConfigFile, err)
	}
	if !exists {
		err := paths.WriteEncryptedFile(configFilePath, passphrase, &store{Wallets: []WalletConfig{}})
		if err != nil {
			return nil, fmt.Errorf("couldn't write default file %s: %w", configFilePath, err)
		}
	}

	walletsFolder, err := vegaPaths.DataDirFor(paths.NodeWalletsDataHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get directory for %s: %w", paths.NodeWalletsDataHome, err)
	}

	return &Loader{
		configFilePath: configFilePath,
		walletsPath:    walletsFolder,
	}, nil
}

func (l *Loader) Load(passphrase string) (*store, error) {
	store := &store{}
	if err := paths.ReadEncryptedFile(l.configFilePath, passphrase, store); err != nil {
		if err.Error() == errInternalWrongPassphrase.Error() {
			return nil, ErrWrongPassphrase
		}
		return nil, fmt.Errorf("couldn't read encrypted file %s: %w", l.configFilePath, err)
	}
	return store, nil
}

func (l *Loader) Save(store *store, passphrase string) error {
	err := paths.WriteEncryptedFile(l.configFilePath, passphrase, store)
	if err != nil {
		return fmt.Errorf("couldn't write encrypted file %s: %w", l.configFilePath, err)
	}
	return nil
}

func (l *Loader) GetConfigFilePath() string {
	return l.configFilePath
}
