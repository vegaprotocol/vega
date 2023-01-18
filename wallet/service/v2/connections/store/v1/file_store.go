package v1

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	vgjob "code.vegaprotocol.io/vega/libs/job"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	"github.com/fsnotify/fsnotify"
)

var ErrStoreNotInitialized = errors.New("the tokens store has not been initialized")

type FileStore struct {
	tokensFilePath string

	passphrase string

	// jobRunner is used to start and stop the file watcher routines.
	jobRunner *vgjob.Runner

	// listeners are callback functions to be called when a change occurs on
	// the tokens.
	listeners []func(context.Context, ...connections.TokenDescription)
	mu        sync.Mutex
}

func (s *FileStore) TokenExists(token connections.Token) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens, err := s.readTokensFile()
	if err != nil {
		return false, err
	}

	tokenStr := token.String()
	for _, tokenInfo := range tokens.Tokens {
		if tokenInfo.Token == tokenStr {
			return true, nil
		}
	}
	return false, nil
}

func (s *FileStore) ListTokens() ([]connections.TokenSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokensFile, err := s.readTokensFile()
	if err != nil {
		return nil, err
	}

	summaries := make([]connections.TokenSummary, 0, len(tokensFile.Tokens))

	for _, tokenInfo := range tokensFile.Tokens {
		token, err := connections.AsToken(tokenInfo.Token)
		if err != nil {
			return nil, fmt.Errorf("token %q is not a valid token: %w", token, err)
		}
		summaries = append(summaries, connections.TokenSummary{
			CreationDate:   tokenInfo.CreationDate,
			Description:    tokenInfo.Description,
			Token:          token,
			ExpirationDate: tokenInfo.ExpirationDate,
		})
	}

	return summaries, nil
}

func (s *FileStore) DescribeToken(token connections.Token) (connections.TokenDescription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens, err := s.readTokensFile()
	if err != nil {
		return connections.TokenDescription{}, err
	}

	tokenStr := token.String()
	for _, tokenInfo := range tokens.Tokens {
		if tokenInfo.Token == tokenStr {
			return connections.TokenDescription{
				Description:    tokenInfo.Description,
				ExpirationDate: tokenInfo.ExpirationDate,
				Token:          token,
				Wallet: connections.WalletCredentials{
					Name:       tokenInfo.Wallet,
					Passphrase: tokens.Resources.Wallets[tokenInfo.Wallet],
				},
			}, nil
		}
	}

	return connections.TokenDescription{}, ErrTokenDoesNotExist
}

func (s *FileStore) SaveToken(token connections.TokenDescription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokensFile, err := s.readTokensFile()
	if err != nil {
		return err
	}

	tokensFile.Resources.Wallets[token.Wallet.Name] = token.Wallet.Passphrase

	tokensFile.Tokens = append(tokensFile.Tokens, tokenContent{
		Token:          token.Token.String(),
		CreationDate:   token.CreationDate,
		Description:    token.Description,
		Wallet:         token.Wallet.Name,
		ExpirationDate: token.ExpirationDate,
	})

	return s.writeTokensFile(tokensFile)
}

func (s *FileStore) DeleteToken(token connections.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens, err := s.readTokensFile()
	if err != nil {
		return err
	}

	tokenStr := token.String()
	walletsInUse := map[string]interface{}{}
	tokensContent := make([]tokenContent, 0, len(tokens.Tokens)-1)
	for _, tokenContent := range tokens.Tokens {
		if tokenStr != tokenContent.Token {
			walletsInUse[tokenContent.Wallet] = nil
			tokensContent = append(tokensContent, tokenContent)
		}
	}
	tokens.Tokens = tokensContent

	wallets := tokens.Resources.Wallets
	for wallet := range wallets {
		if _, ok := walletsInUse[wallet]; !ok {
			delete(tokens.Resources.Wallets, wallet)
		}
	}

	return s.writeTokensFile(tokens)
}

func (s *FileStore) OnUpdate(callbackFn func(ctx context.Context, tokens ...connections.TokenDescription)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listeners = append(s.listeners, callbackFn)
}

func (s *FileStore) Close() {
	if s.jobRunner != nil {
		s.jobRunner.StopAllJobs()
	}
}

func (s *FileStore) readTokensFile() (tokensFile, error) {
	tokens := tokensFile{}

	exists, err := vgfs.FileExists(s.tokensFilePath)
	if err != nil {
		return tokensFile{}, fmt.Errorf("could not verify the existence of the tokens file: %w", err)
	} else if !exists {
		return defaultTokensFileContent(), nil
	}

	if err := paths.ReadEncryptedFile(s.tokensFilePath, s.passphrase, &tokens); err != nil {
		if err.Error() == "couldn't decrypt content: cipher: message authentication failed" {
			return tokensFile{}, api.ErrWrongPassphrase
		}
		return tokensFile{}, fmt.Errorf("couldn't read the file %s: %w", s.tokensFilePath, err)
	}

	if tokens.Resources.Wallets == nil {
		tokens.Resources.Wallets = map[string]string{}
	}

	if tokens.Tokens == nil {
		tokens.Tokens = []tokenContent{}
	}

	return tokens, nil
}

func (s *FileStore) writeTokensFile(tokens tokensFile) error {
	if err := paths.WriteEncryptedFile(s.tokensFilePath, s.passphrase, tokens); err != nil {
		return fmt.Errorf("couldn't write the file %s: %w", s.tokensFilePath, err)
	}

	return nil
}

func (s *FileStore) wipeOut() error {
	exists, err := vgfs.FileExists(s.tokensFilePath)
	if err != nil {
		return fmt.Errorf("could not verify the existence of the tokens file: %w", err)
	}

	if exists {
		if err := os.Remove(s.tokensFilePath); err != nil {
			return fmt.Errorf("could not remove the tokens file: %w", err)
		}
	}

	return nil
}

func (s *FileStore) initDefault() error {
	return s.writeTokensFile(defaultTokensFileContent())
}

func (s *FileStore) startFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not start the token store watcher: %w", err)
	}

	s.jobRunner = vgjob.NewRunner(context.Background())

	s.jobRunner.Go(func(ctx context.Context) {
		s.watchFile(ctx, watcher)
	})

	if err := watcher.Add(s.tokensFilePath); err != nil {
		return fmt.Errorf("could not start watching the token file: %w", err)
	}

	return nil
}

func (s *FileStore) watchFile(ctx context.Context, watcher *fsnotify.Watcher) {
	defer func() {
		_ = watcher.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			s.broadcastFileChanges(ctx, watcher, event)
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
			// Something went wrong, but tons of thing can go wrong on a file
			// system, and there is nothing we can do about that. Let's ignore it.
		}
	}
}

func (s *FileStore) broadcastFileChanges(ctx context.Context, watcher *fsnotify.Watcher, event fsnotify.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event.Op == fsnotify.Chmod {
		// If this is solely a CHMOD, we do not trigger an update.
		return
	}

	if event.Has(fsnotify.Remove) {
		_ = watcher.Remove(s.tokensFilePath)
		exists, err := vgfs.FileExists(s.tokensFilePath)
		if err != nil {
			return
		}
		if !exists {
			// The file could have been re-created before we acquire the
			// lock.
			_ = s.initDefault()
		}
		_ = watcher.Add(s.tokensFilePath)
	}

	// Let's wait a bit so any write actions in progress have a chance to finish.
	// This is far from being resilient, but it can help.
	time.Sleep(100 * time.Millisecond)

	tokenDescriptions, err := s.readFileAsTokenDescriptions()
	if err != nil {
		// This can be the result of concurrent modification on the token file.
		// The best thing to do is to carry on and ignore the changes.
		return
	}

	for _, listener := range s.listeners {
		listener(ctx, tokenDescriptions...)
	}
}

func (s *FileStore) readFileAsTokenDescriptions() ([]connections.TokenDescription, error) {
	exists, err := vgfs.FileExists(s.tokensFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not verify the token file existence: %w", err)
	}
	if !exists {
		// The file got deleted.
		//
		// This can the result of a "desperate" action to kill all long-living
		// connections, using the `api-token init --force` command.
		//
		// As a result, we return an empty list of tokens, meaning, all tokens
		// should be considered invalid from now on.
		return nil, nil
	}

	tokensFile, err := s.readTokensFile()
	if err != nil {
		return nil, fmt.Errorf("could not read the token file: %w", err)
	}

	tokenDescriptions := make([]connections.TokenDescription, 0, len(tokensFile.Tokens))

	for _, tokenInfo := range tokensFile.Tokens {
		token, err := connections.AsToken(tokenInfo.Token)
		if err != nil {
			// It's all or nothing.
			return nil, fmt.Errorf("the token %q could not be parse: %w", tokenInfo.Token, err)
		}
		tokenDescriptions = append(tokenDescriptions, connections.TokenDescription{
			Description:    tokenInfo.Description,
			CreationDate:   tokenInfo.CreationDate,
			ExpirationDate: tokenInfo.ExpirationDate,
			Token:          token,
			Wallet: connections.WalletCredentials{
				Name:       tokenInfo.Wallet,
				Passphrase: tokensFile.Resources.Wallets[tokenInfo.Wallet],
			},
		})
	}

	return tokenDescriptions, nil
}

func InitialiseStore(p paths.Paths, passphrase string) (*FileStore, error) {
	tokensFilePath, err := p.CreateDataPathFor(paths.WalletServiceTokensDataFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get data path for %s: %w", paths.WalletServicePublicRSAKeyDataFile, err)
	}

	store := &FileStore{
		tokensFilePath: tokensFilePath,
		passphrase:     passphrase,
		listeners:      []func(context.Context, ...connections.TokenDescription){},
	}

	exists, err := vgfs.FileExists(tokensFilePath)
	if err != nil || !exists {
		return nil, ErrStoreNotInitialized
	}

	if _, err := store.readTokensFile(); err != nil {
		return nil, err
	}

	if err := store.startFileWatcher(); err != nil {
		store.Close()
		return nil, err
	}

	return store, nil
}

func ReinitialiseStore(p paths.Paths, passphrase string) (*FileStore, error) {
	tokensFilePath, err := tokensFilePath(p)
	if err != nil {
		return nil, err
	}

	store := &FileStore{
		tokensFilePath: tokensFilePath,
		passphrase:     passphrase,
		listeners:      []func(context.Context, ...connections.TokenDescription){},
	}

	if err := store.wipeOut(); err != nil {
		return nil, err
	}

	if err := store.initDefault(); err != nil {
		return nil, err
	}

	if err := store.startFileWatcher(); err != nil {
		store.Close()
		return nil, err
	}

	return store, nil
}

func IsStoreBootstrapped(p paths.Paths) (bool, error) {
	tokensFilePath, err := tokensFilePath(p)
	if err != nil {
		return false, err
	}

	exists, err := vgfs.FileExists(tokensFilePath)

	return err == nil && exists, nil
}

func tokensFilePath(p paths.Paths) (string, error) {
	tokensFilePath, err := p.CreateDataPathFor(paths.WalletServiceTokensDataFile)
	if err != nil {
		return "", fmt.Errorf("couldn't get data path for %s: %w", paths.WalletServicePublicRSAKeyDataFile, err)
	}
	return tokensFilePath, nil
}

type tokensFile struct {
	FileVersion   int              `json:"fileVersion"`
	TokensVersion int              `json:"tokensVersion"`
	Resources     resourcesContent `json:"resources"`
	Tokens        []tokenContent   `json:"tokens"`
}

func defaultTokensFileContent() tokensFile {
	return tokensFile{
		FileVersion:   1,
		TokensVersion: 1,
		Resources: resourcesContent{
			Wallets: map[string]string{},
		},
		Tokens: []tokenContent{},
	}
}

type resourcesContent struct {
	Wallets map[string]string `json:"wallets"`
}

type tokenContent struct {
	Token          string     `json:"token"`
	CreationDate   time.Time  `json:"creationDate"`
	ExpirationDate *time.Time `json:"expirationDate"`
	Description    string     `json:"description"`
	Wallet         string     `json:"wallet"`
}
