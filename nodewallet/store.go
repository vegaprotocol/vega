package nodewallet

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	vgcrypto "code.vegaprotocol.io/shared/libs/crypto"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
)

const (
	defaultStoreFile = "store"
	nodeWalletFolder = "nodewallet"
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

type storage struct {
	storePath   string
	walletsPath string
}

func Initialise(rootPath, passphrase string) error {
	storage := newStorage(rootPath)
	return storage.Initialise(passphrase)
}

func newStorage(rootPath string) *storage {
	return &storage{
		storePath:   filepath.Join(rootPath, nodeWalletFolder, defaultStoreFile),
		walletsPath: filepath.Join(rootPath, nodeWalletFolder),
	}
}

func (s *storage) Initialise(passphrase string) error {
	err := vgfs.EnsureDir(s.walletsPath)
	if err != nil {
		return err
	}

	exists, err := vgfs.FileExists(s.storePath)
	if err != nil {
		if _, ok := err.(*vgfs.PathNotFound); !ok {
			return err
		}
	}
	if !exists {
		return s.Save(&store{Wallets: []WalletConfig{}}, passphrase)
	}
	return nil
}

func (s *storage) WalletDirFor(name Blockchain) string {
	return filepath.Join(s.walletsPath, strings.ToLower(string(name)))
}

func (s *storage) Load(passphrase string) (*store, error) {
	if ok, err := vgfs.PathExists(s.storePath); !ok {
		return nil, fmt.Errorf("unable to load store (%v)", err)
	}

	data, err := vgfs.ReadFile(s.storePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read store file (%v)", err)
	}

	decBuf, err := vgcrypto.Decrypt(data, passphrase)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt store file (%v)", err)
	}

	store := &store{}
	return store, json.Unmarshal(decBuf, store)
}

func (s *storage) Save(store *store, passphrase string) error {
	buf, err := json.Marshal(store)
	if err != nil {
		return err
	}

	encBuf, err := vgcrypto.Encrypt(buf, passphrase)
	if err != nil {
		return fmt.Errorf("unable to encrypt store file (%v)", err)
	}

	return vgfs.WriteFile(s.storePath, encBuf)
}
