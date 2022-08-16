package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

type Store struct {
	walletsHome string
}

func InitialiseStore(walletsHome string) (*Store, error) {
	if err := vgfs.EnsureDir(walletsHome); err != nil {
		return nil, fmt.Errorf("couldn't ensure directories at %s: %w", walletsHome, err)
	}

	return &Store{
		walletsHome: walletsHome,
	}, nil
}

func (s *Store) DeleteWallet(ctx context.Context, name string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	walletPath := s.walletPath(name)

	if exists, _ := vgfs.PathExists(walletPath); !exists {
		return wallet.ErrWalletDoesNotExists
	}

	if err := os.Remove(walletPath); err != nil {
		return fmt.Errorf("couldn't remove wallet file at %s: %w", walletPath, err)
	}
	return nil
}

func (s *Store) WalletExists(ctx context.Context, name string) (bool, error) {
	if err := checkContextStatus(ctx); err != nil {
		return false, err
	}

	walletPath := s.walletPath(name)

	exists, err := vgfs.PathExists(walletPath)
	if err != nil {
		return false, fmt.Errorf("couldn't verify path: %w", err)
	}
	return exists, err
}

func (s *Store) ListWallets(ctx context.Context) ([]string, error) {
	if err := checkContextStatus(ctx); err != nil {
		return nil, err
	}

	walletsParentDir, walletsDir := filepath.Split(s.walletsHome)
	entries, err := fs.ReadDir(os.DirFS(walletsParentDir), walletsDir)
	if err != nil {
		return nil, fmt.Errorf("couldn't read directory at %s: %w", s.walletsHome, err)
	}
	wallets := make([]string, len(entries))
	for i, entry := range entries {
		wallets[i] = entry.Name()
	}
	sort.Strings(wallets)
	return wallets, nil
}

func (s *Store) GetWallet(ctx context.Context, name, passphrase string) (wallet.Wallet, error) {
	if err := checkContextStatus(ctx); err != nil {
		return nil, err
	}

	buf, err := fs.ReadFile(os.DirFS(s.walletsHome), name)
	if err != nil {
		return nil, fmt.Errorf("couldn't read file at %s: %w", s.walletsHome, err)
	}

	decBuf, err := vgcrypto.Decrypt(buf, passphrase)
	if err != nil {
		if err.Error() == "cipher: message authentication failed" {
			return nil, wallet.ErrWrongPassphrase
		}
		return nil, err
	}

	versionedWallet := &struct {
		Version uint32 `json:"version"`
	}{}

	if err := json.Unmarshal(decBuf, versionedWallet); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal wallet verion: %w", err)
	}

	if !wallet.IsVersionSupported(versionedWallet.Version) {
		return nil, wallet.NewUnsupportedWalletVersionError(versionedWallet.Version)
	}

	w := &wallet.HDWallet{}
	if err := json.Unmarshal(decBuf, w); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal wallet: %w", err)
	}

	// The wallet name is not saved in the file to avoid de-synchronisation
	// between file name and file content
	w.SetName(name)

	return w, nil
}

func (s *Store) SaveWallet(ctx context.Context, w wallet.Wallet, passphrase string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	buf, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("couldn't marshal wallet: %w", err)
	}

	encBuf, err := vgcrypto.Encrypt(buf, passphrase)
	if err != nil {
		return fmt.Errorf("couldn't encrypt wallet: %w", err)
	}

	walletPath := s.walletPath(w.Name())
	err = vgfs.WriteFile(walletPath, encBuf)
	if err != nil {
		return fmt.Errorf("couldn't write wallet file at %s: %w", walletPath, err)
	}
	return nil
}

func (s *Store) GetWalletPath(name string) string {
	return s.walletPath(name)
}

func (s *Store) walletPath(name string) string {
	return filepath.Join(s.walletsHome, name)
}

func checkContextStatus(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
