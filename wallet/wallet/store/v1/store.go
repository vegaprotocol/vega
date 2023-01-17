package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

var (
	ErrWalletNameCannotContainSlashCharacters = errors.New("the name cannot contain slash (\"/\", \"\\\") characters")
	ErrWalletNameCannotStartWithDot           = errors.New("the name cannot start with a dot (\".\") character")
)

type Store struct {
	// walletsHome is the path to the folder containing all the wallet file.
	walletsHome string

	// mu is a mutex ensuring there is no concurrent access to the wallets
	// since this store is used by jobs ran in parallel.
	mu sync.Mutex

	// unlockedWallets maps wallet fingerprint to an unlocked wallet.
	//
	// WARNING: This implementation has a major drawback. It does not handle the
	// 	renaming of a wallet shared by different jobs. If the wallet has been
	// 	renamed by a job, while being used by another, any call to methods using
	// 	the previous name will either fail, or return the previous instance of
	// 	the wallet, or regenerate the wallet file with the previous name.
	unlockedWallets map[fingerprint]*unlockedWallet

	// listeners are callback functions to be called when a wallet update.
	listeners []func(context.Context, wallet.Wallet)
}

type fingerprint string

type unlockedWallet struct {
	passphrase string
	wallet     wallet.Wallet
}

func InitialiseStore(walletsHome string) (*Store, error) {
	if err := vgfs.EnsureDir(walletsHome); err != nil {
		return nil, fmt.Errorf("could not ensure directories at %s: %w", walletsHome, err)
	}

	return &Store{
		walletsHome:     walletsHome,
		unlockedWallets: map[fingerprint]*unlockedWallet{},
		listeners:       []func(context.Context, wallet.Wallet){},
	}, nil
}

func (s *Store) UnlockWallet(ctx context.Context, name, passphrase string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check whether the wallet is already unlocked, or not.
	for _, uw := range s.unlockedWallets {
		if uw.wallet.Name() == name {
			// The wallet is already unlocked, but we still need to verify the
			// passphrase correctness.
			if uw.passphrase == passphrase {
				// Everything's fine.
				return nil
			}
			return wallet.ErrWrongPassphrase
		}
	}

	buf, err := fs.ReadFile(os.DirFS(s.walletsHome), name)
	if err != nil {
		return fmt.Errorf("could not read the wallet file at %s: %w", s.walletsHome, err)
	}

	decBuf, err := vgcrypto.Decrypt(buf, passphrase)
	if err != nil {
		if err.Error() == "cipher: message authentication failed" {
			return wallet.ErrWrongPassphrase
		}
		return err
	}

	versionedWallet := &struct {
		Version uint32 `json:"version"`
	}{}

	if err := json.Unmarshal(decBuf, versionedWallet); err != nil {
		return fmt.Errorf("could not unmarshal the wallet version: %w", err)
	}

	if !wallet.IsKeyDerivationVersionSupported(versionedWallet.Version) {
		return wallet.NewUnsupportedWalletVersionError(versionedWallet.Version)
	}

	w := &wallet.HDWallet{}
	if err := json.Unmarshal(decBuf, w); err != nil {
		return fmt.Errorf("could not unmarshal the wallet: %w", err)
	}

	// The wallet name is not saved in the file to avoid de-synchronisation
	// between file name and file content. We use the filename to set the
	// wallet name. This allows users to rename the wallet file without fear of
	// broken state.
	w.SetName(name)

	s.unlockedWallets[asFingerprint(w)] = &unlockedWallet{
		passphrase: passphrase,
		wallet:     w,
	}

	return nil
}

func (s *Store) LockWallet(ctx context.Context, name string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lockWalletIfUnlocked(name)

	return nil
}

// DeleteWallet deletes the wallet file in place.
// It does not require the wallets to be unlocked, but lock it if so.
func (s *Store) DeleteWallet(ctx context.Context, name string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lockWalletIfUnlocked(name)

	walletPath := s.walletPath(name)

	if exists, err := vgfs.PathExists(walletPath); err != nil {
		return fmt.Errorf("could not verify the path at %s: %w", walletPath, err)
	} else if !exists {
		return api.ErrWalletDoesNotExist
	}

	return os.Remove(walletPath)
}

// UpdatePassphrase update the passphrase used to encrypt the wallet.
// It requires the wallet to be unlocked.
func (s *Store) UpdatePassphrase(ctx context.Context, name, newPassphrase string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	walletPath := s.walletPath(name)

	if exists, err := vgfs.PathExists(walletPath); err != nil {
		return fmt.Errorf("could not verify the path at %s: %w", walletPath, err)
	} else if !exists {
		return api.ErrWalletDoesNotExist
	}

	for _, uw := range s.unlockedWallets {
		if uw.wallet.Name() == name {
			if err := s.writeWallet(uw.wallet, newPassphrase); err != nil {
				return err
			}
			uw.passphrase = newPassphrase
			return nil
		}
	}

	return api.ErrWalletIsLocked
}

// RenameWallet renames a wallet file in place.
// It does not require the wallets to be unlocked, but updates the unlocked wallet
// if so.
func (s *Store) RenameWallet(ctx context.Context, currentName, newName string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	if err := ensureValidWalletName(newName); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	currentWalletPath := s.walletPath(currentName)

	if exists, err := vgfs.PathExists(currentWalletPath); err != nil {
		return fmt.Errorf("could not verify the path at %s: %w", currentWalletPath, err)
	} else if !exists {
		return api.ErrWalletDoesNotExist
	}

	newWalletPath := s.walletPath(newName)

	if err := os.Rename(currentWalletPath, newWalletPath); err != nil {
		return fmt.Errorf("could not rename the wallet %q to %q at %s: %w", currentName, newName, s.walletsHome, err)
	}

	for fingerprint, uw := range s.unlockedWallets {
		if uw.wallet.Name() == currentName {
			uw.wallet.SetName(newName)
			// Update the fingerprint with changed name.
			s.unlockedWallets[asFingerprint(uw.wallet)] = uw
			delete(s.unlockedWallets, fingerprint)
			return nil
		}
	}

	return nil
}

// WalletExists verify if file matching the name exist locally.
// It does not require the wallet to be unlocked.
// It does not ensure the file is an actual wallet.
func (s *Store) WalletExists(ctx context.Context, name string) (bool, error) {
	if err := checkContextStatus(ctx); err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	walletPath := s.walletPath(name)

	exists, err := vgfs.PathExists(walletPath)
	if err != nil {
		return false, fmt.Errorf("could not verify the path at %s: %w", walletPath, err)
	}
	return exists, nil
}

// ListWallets list all existing wallets stored locally.
// It does not require the wallets to be unlocked.
// It assumes that all the file under the "walletHome" are wallets. It does not
// ensure the files are actual wallets.
// Hidden files are excluded.
func (s *Store) ListWallets(ctx context.Context) ([]string, error) {
	if err := checkContextStatus(ctx); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	walletsParentDir, walletsDir := filepath.Split(s.walletsHome)
	entries, err := fs.ReadDir(os.DirFS(walletsParentDir), walletsDir)
	if err != nil {
		return nil, fmt.Errorf("could not read the directory at %s: %w", s.walletsHome, err)
	}
	wallets := []string{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			continue
		}

		wallets = append(wallets, entry.Name())
	}
	sort.Strings(wallets)
	return wallets, nil
}

// GetWallet requires the wallet to be unlocked first, using Store.UnlockWallet().
func (s *Store) GetWallet(ctx context.Context, name string) (wallet.Wallet, error) {
	if err := checkContextStatus(ctx); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	walletPath := s.walletPath(name)

	if exists, err := vgfs.PathExists(walletPath); err != nil {
		return nil, fmt.Errorf("could not verify the path at %s: %w", walletPath, err)
	} else if !exists {
		return nil, api.ErrWalletDoesNotExist
	}

	for _, uw := range s.unlockedWallets {
		if uw.wallet.Name() == name {
			return uw.wallet.Clone(), nil
		}
	}

	return nil, api.ErrWalletIsLocked
}

// CreateWallet creates a wallet, and automatically load it as an unlocked wallet.
func (s *Store) CreateWallet(ctx context.Context, w wallet.Wallet, passphrase string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.writeWallet(w, passphrase); err != nil {
		return err
	}

	s.unlockedWallets[asFingerprint(w)] = &unlockedWallet{
		passphrase: passphrase,
		wallet:     w.Clone(),
	}

	return nil
}

// UpdateWallet updates an unlocked wallet.
// If this method is called with a wallet that had the name changed, a new file
// is written and the previous one is not deleted. To rename the wallet in-place,
// the method Store.RenameWallet() should be used instead.
func (s *Store) UpdateWallet(ctx context.Context, w wallet.Wallet) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	uw, unlocked := s.unlockedWallets[asFingerprint(w)]
	if !unlocked {
		return api.ErrWalletIsLocked
	}

	if err := s.writeWallet(w, uw.passphrase); err != nil {
		return err
	}

	// At this point, we no longer have point of failure, so we update the wallet
	// reference.
	uw.wallet = w.Clone()

	for _, listener := range s.listeners {
		listener(ctx, w.Clone())
	}

	return nil
}

func (s *Store) OnUpdate(callbackFn func(context.Context, wallet.Wallet)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listeners = append(s.listeners, callbackFn)
}

func (s *Store) GetWalletPath(name string) string {
	return s.walletPath(name)
}

func (s *Store) writeWallet(w wallet.Wallet, passphrase string) error {
	if err := ensureValidWalletName(w.Name()); err != nil {
		return err
	}

	buf, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("could not marshal wallet: %w", err)
	}

	encBuf, err := vgcrypto.Encrypt(buf, passphrase)
	if err != nil {
		return fmt.Errorf("could not encrypt wallet: %w", err)
	}

	walletPath := s.walletPath(w.Name())
	err = vgfs.WriteFile(walletPath, encBuf)
	if err != nil {
		return fmt.Errorf("could not write wallet file at %s: %w", walletPath, err)
	}

	return nil
}

func (s *Store) walletPath(name string) string {
	return filepath.Join(s.walletsHome, name)
}

func (s *Store) lockWalletIfUnlocked(name string) {
	for walletFingerprint, uw := range s.unlockedWallets {
		if uw.wallet.Name() == name {
			uw.wallet = nil
			delete(s.unlockedWallets, walletFingerprint)
			return
		}
	}
}

func checkContextStatus(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func asFingerprint(w wallet.Wallet) fingerprint {
	dataToFingerprint := fmt.Sprintf("%s::%d::%s", w.ID(), w.KeyDerivationVersion(), w.Name())
	strFingerprint := vgcrypto.HashStrToHex(dataToFingerprint)
	return fingerprint(strFingerprint)
}

func ensureValidWalletName(newName string) error {
	// Reject hidden files.
	if strings.HasPrefix(newName, ".") {
		return ErrWalletNameCannotStartWithDot
	}

	// Reject slash and back-slash to avoid path resolution.
	if strings.ContainsAny(newName, "/\\") {
		return ErrWalletNameCannotContainSlashCharacters
	}

	return nil
}
