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
	vgjob "code.vegaprotocol.io/vega/libs/job"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/fsnotify/fsnotify"
)

var (
	ErrWalletNameCannotContainSlashCharacters = errors.New("the name cannot contain slash (\"/\", \"\\\") characters")
	ErrWalletNameCannotStartWithDot           = errors.New("the name cannot start with a dot (\".\") character")
	ErrWalletFileIsEmpty                      = errors.New("the wallet file is empty")
)

type FileStore struct {
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
	unlockedWallets map[string]*unlockedWallet

	// listeners are callback functions to be called when a wallet update.
	listeners []func(context.Context, wallet.Event)

	watcher   *fsnotify.Watcher
	jobRunner *vgjob.Runner
}

type unlockedWallet struct {
	passphrase string
	wallet     wallet.Wallet
}

func InitialiseStore(walletsHome string, withFileWatcher bool) (*FileStore, error) {
	if err := vgfs.EnsureDir(walletsHome); err != nil {
		return nil, fmt.Errorf("could not ensure directories at %s: %w", walletsHome, err)
	}

	store := &FileStore{
		walletsHome:     walletsHome,
		unlockedWallets: map[string]*unlockedWallet{},
		listeners:       []func(context.Context, wallet.Event){},
	}

	if withFileWatcher {
		if err := store.startFilesWatcher(); err != nil {
			store.Close()
			return nil, err
		}
	}

	return store, nil
}

func (s *FileStore) UnlockWallet(ctx context.Context, name, passphrase string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check whether the wallet is already unlocked, or not.
	uw, isUnlocked := s.unlockedWallets[name]
	if isUnlocked {
		if uw.passphrase == passphrase {
			// Credentials are fine, and wallet is already unlocked.
			return nil
		}
		return wallet.ErrWrongPassphrase
	}

	w, err := s.readWalletFile(name, passphrase)
	if err != nil {
		return err
	}

	s.unlockedWallets[w.Name()] = &unlockedWallet{
		passphrase: passphrase,
		wallet:     w,
	}

	// It warns other components of a freshly unlocked wallet.
	s.broadcastEvent(ctx, wallet.NewUnlockedWalletUpdatedEvent(w))

	return nil
}

func (s *FileStore) IsWalletAlreadyUnlocked(ctx context.Context, name string) (bool, error) {
	if err := checkContextStatus(ctx); err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, isUnlocked := s.unlockedWallets[name]
	return isUnlocked, nil
}

func (s *FileStore) LockWallet(ctx context.Context, name string) error {
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
func (s *FileStore) DeleteWallet(ctx context.Context, name string) error {
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
func (s *FileStore) UpdatePassphrase(ctx context.Context, name, newPassphrase string) error {
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

	uw, isUnlocked := s.unlockedWallets[name]
	if !isUnlocked {
		return api.ErrWalletIsLocked
	}

	if err := s.writeWallet(uw.wallet, newPassphrase); err != nil {
		return err
	}
	uw.passphrase = newPassphrase
	return nil
}

// RenameWallet renames a wallet file in place.
// It does not require the wallets to be unlocked, but updates the unlocked wallet
// if so.
func (s *FileStore) RenameWallet(ctx context.Context, currentName, newName string) error {
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

	uw, isUnlocked := s.unlockedWallets[currentName]
	if isUnlocked {
		uw.wallet.SetName(newName)
		delete(s.unlockedWallets, currentName)
		s.unlockedWallets[newName] = uw
	}

	s.broadcastEvent(ctx, wallet.NewWalletRenamedEvent(currentName, newName))

	return nil
}

// WalletExists verify if file matching the name exist locally.
// It does not require the wallet to be unlocked.
// It does not ensure the file is an actual wallet.
func (s *FileStore) WalletExists(ctx context.Context, name string) (bool, error) {
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
func (s *FileStore) ListWallets(ctx context.Context) ([]string, error) {
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
	wallets := make([]string, 0, len(entries))
	walletCount := 0
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			continue
		}

		wallets = append(wallets, entry.Name())
		walletCount++
	}
	wallets = wallets[0:walletCount]
	sort.Strings(wallets)
	return wallets, nil
}

// GetWallet requires the wallet to be unlocked first, using FileStore.UnlockWallet().
func (s *FileStore) GetWallet(ctx context.Context, name string) (wallet.Wallet, error) {
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

	uw, isUnlocked := s.unlockedWallets[name]
	if !isUnlocked {
		return nil, api.ErrWalletIsLocked
	}

	return uw.wallet.Clone(), nil
}

// CreateWallet creates a wallet, and automatically load it as an unlocked wallet.
func (s *FileStore) CreateWallet(ctx context.Context, w wallet.Wallet, passphrase string) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.writeWallet(w, passphrase); err != nil {
		return err
	}

	s.unlockedWallets[w.Name()] = &unlockedWallet{
		passphrase: passphrase,
		wallet:     w.Clone(),
	}

	return nil
}

// UpdateWallet updates an unlocked wallet.
// If this method is called with a wallet that had the name changed, a new file
// is written and the previous one is not deleted. To rename the wallet in-place,
// the method FileStore.RenameWallet() should be used instead.
func (s *FileStore) UpdateWallet(ctx context.Context, w wallet.Wallet) error {
	if err := checkContextStatus(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	uw, isUnlocked := s.unlockedWallets[w.Name()]
	if !isUnlocked {
		return api.ErrWalletIsLocked
	}

	if err := s.writeWallet(w, uw.passphrase); err != nil {
		return err
	}

	// At this point, we no longer have point of failure, so we update the wallet
	// reference.
	uw.wallet = w.Clone()

	return nil
}

func (s *FileStore) OnUpdate(callbackFn func(context.Context, wallet.Event)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listeners = append(s.listeners, callbackFn)
}

func (s *FileStore) Close() {
	if s.jobRunner != nil {
		s.jobRunner.StopAllJobs()
	}
}

func (s *FileStore) GetWalletPath(name string) string {
	return s.walletPath(name)
}

func (s *FileStore) writeWallet(w wallet.Wallet, passphrase string) (rerr error) {
	defer func() {
		if r := recover(); r != nil {
			rerr = fmt.Errorf("a system error occurred while writing the wallet file: %s", r)
		}
	}()
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

func (s *FileStore) walletPath(name string) string {
	return filepath.Join(s.walletsHome, name)
}

func (s *FileStore) readWalletFile(name string, passphrase string) (hd *wallet.HDWallet, rerr error) {
	defer func() {
		if r := recover(); r != nil {
			hd, rerr = nil, fmt.Errorf("a system error occurred while reading the wallet file: %s", r)
		}
	}()

	buf, err := fs.ReadFile(os.DirFS(s.walletsHome), name)
	if err != nil {
		return nil, fmt.Errorf("could not read the wallet file at %s: %w", s.walletsHome, err)
	}

	if len(buf) == 0 {
		return nil, ErrWalletFileIsEmpty
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
		return nil, fmt.Errorf("could not unmarshal the wallet version: %w", err)
	}

	if !wallet.IsKeyDerivationVersionSupported(versionedWallet.Version) {
		return nil, wallet.NewUnsupportedWalletVersionError(versionedWallet.Version)
	}

	w := &wallet.HDWallet{}
	if err := json.Unmarshal(decBuf, w); err != nil {
		return nil, fmt.Errorf("could not unmarshal the wallet: %w", err)
	}

	// The wallet name is not saved in the file to avoid de-synchronisation
	// between file name and file content. We use the filename to set the
	// wallet name. This allows users to rename the wallet file without fear of
	// broken state.
	w.SetName(name)

	return w, nil
}

func (s *FileStore) lockWalletIfUnlocked(name string) {
	uw, isUnlocked := s.unlockedWallets[name]
	if !isUnlocked {
		return
	}

	uw.wallet = nil
	delete(s.unlockedWallets, name)
}

func (s *FileStore) startFilesWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not start the wallets watcher: %w", err)
	}

	s.watcher = watcher

	s.jobRunner = vgjob.NewRunner(context.Background())

	s.jobRunner.Go(func(ctx context.Context) {
		s.watchFile(ctx)
	})

	if err := s.watcher.Add(s.walletsHome); err != nil {
		return fmt.Errorf("could not start watching the wallet files: %w", err)
	}

	return nil
}

func (s *FileStore) watchFile(watcherCtx context.Context) {
	defer func() {
		_ = s.watcher.Close()
	}()

	for {
		select {
		case <-watcherCtx.Done():
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			s.applyFileChangesToStore(watcherCtx, event)
		case _, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			// Something went wrong, but tons of thing can go wrong on a file
			// system, and there is nothing we can do about that. Let's ignore it.
		}
	}
}

func (s *FileStore) applyFileChangesToStore(ctx context.Context, event fsnotify.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event.Op == fsnotify.Chmod {
		// If this is solely a CHMOD, we do not trigger an update.
		return
	}

	pathPrefixCount := len(s.walletsHome)

	if len(event.Name) <= pathPrefixCount {
		// An event targeting the wallets folder has been raised. We don't
		// handle cases that target the whole folder, only individual wallet file.
		return
	}

	walletName := event.Name[pathPrefixCount+1:]

	if err := ensureValidWalletName(walletName); err != nil {
		// The change detected in the folder doesn't look like a wallet name, so
		// that could be anything. Let's ignore it.
		return
	}

	// This doesn't handle wallet files being externally renamed because the
	// fsnotify library doesn't properly support it:
	//
	//     https://github.com/fsnotify/fsnotify/issues/26
	//
	// The event RENAME and CREATE can be raised in arbitrary order, which
	// can work in the RENAME-CREATE order, but hardly in the CREATE-RENAME
	// one, because we don't know if the CREATE is an actual creation or if it's
	// the result of a renaming.
	//
	// We only support renaming made from the same process as the listeners.
	// If made in another process this will not be properly propagated.
	if event.Has(fsnotify.Remove) {
		s.handleWalletFileRemoval(ctx, walletName)
	} else if event.Has(fsnotify.Create) {
		s.broadcastEvent(ctx, wallet.NewWalletCreatedEvent(walletName))
	} else if event.Has(fsnotify.Write) {
		s.handleWalletFileUpdate(ctx, walletName)
	}
}

func (s *FileStore) handleWalletFileUpdate(ctx context.Context, walletName string) {
	uw, isUnlocked := s.unlockedWallets[walletName]
	if !isUnlocked {
		s.broadcastEvent(ctx, wallet.NewLockedWalletUpdateEvent(walletName))
		return
	}

	updatedWallet, err := s.readWalletFile(uw.wallet.Name(), uw.passphrase)
	if err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			// The passphrase changed externally, so we lock the wallet...
			delete(s.unlockedWallets, uw.wallet.Name())
			// ... and treat it like a wallet that has been locked.
			s.broadcastEvent(ctx, wallet.NewWalletHasBeenLockedEvent(uw.wallet.Name()))
		}
		// If we end up here, it means the file couldn't be read. Skipping.
		return
	}

	uw.wallet = updatedWallet

	s.broadcastEvent(ctx, wallet.NewUnlockedWalletUpdatedEvent(updatedWallet))
}

func (s *FileStore) handleWalletFileRemoval(ctx context.Context, walletName string) {
	_, isUnlocked := s.unlockedWallets[walletName]
	if isUnlocked {
		delete(s.unlockedWallets, walletName)
	}
	s.broadcastEvent(ctx, wallet.NewWalletRemovedEvent(walletName))
}

func (s *FileStore) broadcastEvent(ctx context.Context, eventToBroadcast wallet.Event) {
	// We start a goroutine to avoid a deadlock if the listeners query the
	// store, when called, because of the mutex.
	go func() {
		for _, listener := range s.listeners {
			listener(ctx, eventToBroadcast)
		}
	}()
}

func checkContextStatus(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
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
