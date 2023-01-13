package v1

import (
	"errors"
	"fmt"
	"os"
	"time"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
)

var ErrStoreNotInitialized = errors.New("the tokens store has not been initialized")

type Store struct {
	tokensFilePath string
	passphrase     string
}

func (s *Store) TokenExists(token connections.Token) (bool, error) {
	tokens, err := s.readFile()
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

func (s *Store) ListTokens() ([]connections.TokenSummary, error) {
	tokens, err := s.readFile()
	if err != nil {
		return nil, err
	}

	summaries := make([]connections.TokenSummary, 0, len(tokens.Tokens))

	for _, tokenInfo := range tokens.Tokens {
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

func (s *Store) DescribeToken(token connections.Token) (connections.TokenDescription, error) {
	tokens, err := s.readFile()
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

func (s *Store) SaveToken(token connections.TokenDescription) error {
	tokens, err := s.readFile()
	if err != nil {
		return err
	}

	tokens.Resources.Wallets[token.Wallet.Name] = token.Wallet.Passphrase

	tokens.Tokens = append(tokens.Tokens, tokenContent{
		Token:          token.Token.String(),
		CreationDate:   time.Now(),
		Description:    token.Description,
		Wallet:         token.Wallet.Name,
		ExpirationDate: token.ExpirationDate,
	})

	return s.writeFile(tokens)
}

func (s *Store) DeleteToken(token connections.Token) error {
	tokens, err := s.readFile()
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

	return s.writeFile(tokens)
}

func (s *Store) readFile() (tokensFile, error) {
	tokens := tokensFile{}

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

func (s *Store) writeFile(tokens tokensFile) error {
	if err := paths.WriteEncryptedFile(s.tokensFilePath, s.passphrase, tokens); err != nil {
		return fmt.Errorf("couldn't write the file %s: %w", s.tokensFilePath, err)
	}

	return nil
}

func (s *Store) wipeOut() error {
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

func (s *Store) initDefault() error {
	return s.writeFile(defaultTokensFileContent())
}

func LoadStore(p paths.Paths, passphrase string) (*Store, error) {
	tokensFilePath, err := p.CreateDataPathFor(paths.WalletServiceTokensDataFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get data path for %s: %w", paths.WalletServicePublicRSAKeyDataFile, err)
	}

	store := &Store{
		tokensFilePath: tokensFilePath,
		passphrase:     passphrase,
	}

	exists, err := vgfs.FileExists(tokensFilePath)
	if err != nil || !exists {
		return nil, ErrStoreNotInitialized
	}

	if _, err := store.readFile(); err != nil {
		return nil, err
	}

	return store, nil
}

func IsStoreInitialized(p paths.Paths) (bool, error) {
	tokensFilePath, err := tokensFilePath(p)
	if err != nil {
		return false, err
	}

	exists, err := vgfs.FileExists(tokensFilePath)

	return err == nil && exists, nil
}

func InitializeStore(p paths.Paths, passphrase string) (*Store, error) {
	tokensFilePath, err := tokensFilePath(p)
	if err != nil {
		return nil, err
	}

	store := &Store{
		tokensFilePath: tokensFilePath,
		passphrase:     passphrase,
	}

	if err := store.wipeOut(); err != nil {
		return nil, err
	}

	if err := store.initDefault(); err != nil {
		return nil, err
	}

	return store, nil
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
