package connections

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

// Manager holds the opened connections between the third-party
// applications and the wallets.
type Manager struct {
	// tokenToConnection maps the token to the connection. It is the base
	// registry for all opened connections.
	tokenToConnection map[Token]*walletConnection

	// sessionFingerprintToToken maps the session fingerprint (wallet + hostname
	// on a session connection). This is used to determine whether a session
	// connection already exists for a given wallet and hostname.
	// It only holds sessions fingerprints.
	// Long-living connections are not tracked.
	sessionFingerprintToToken map[string]Token

	// walletToTokens maps the wallet name to all the tokens used in connections.
	// This is used to easily retrieve of all the connections made to a given
	// wallet.
	// It holds long-living and session connections.
	walletToTokens map[string][]Token

	// timeService is used to resolve the current time to update the last activity
	// time on the token, and figure out their expiration.
	timeService TimeService

	walletStore WalletStore

	tokenStore TokenStore

	mu sync.Mutex
}

type walletConnection struct {
	// connectedWallet is the projection of the wallet through the permissions
	// and authentication system. On a regular wallet, there are no restrictions
	// on what we can call, which doesn't fit the model of having restricted
	// access, so we wrap the "regular wallet" behind the "connected wallet".
	connectedWallet api.ConnectedWallet

	policy connectionPolicy
}

// StartSession initializes a connection between a wallet and a third-party
// application.
// If a connection already exists, it's disconnected and a new token is
// generated.
func (m *Manager) StartSession(hostname string, w wallet.Wallet) (Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, alreadyLoaded := m.walletToTokens[w.Name()]
	if !alreadyLoaded {
		m.walletToTokens[w.Name()] = []Token{}
	}

	cw, err := api.NewConnectedWallet(hostname, w)
	if err != nil {
		return "", fmt.Errorf("could not instantiate the connected wallet for a session connection: %w", err)
	}

	newToken := m.generateToken()

	sessionFingerprint := asSessionFingerprint(hostname, w.Name())
	if previousToken, sessionAlreadyExists := m.sessionFingerprintToToken[sessionFingerprint]; sessionAlreadyExists {
		m.destroySessionToken(previousToken)
	}
	m.sessionFingerprintToToken[sessionFingerprint] = newToken

	m.walletToTokens[w.Name()] = append(m.walletToTokens[w.Name()], newToken)

	m.tokenToConnection[newToken] = &walletConnection{
		connectedWallet: cw,
		policy: &sessionPolicy{
			lastActivityDate: m.timeService.Now(),
		},
	}

	return newToken, nil
}

func (m *Manager) EndSessionConnectionWithToken(token Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.destroySessionToken(token)

	return nil
}

func (m *Manager) EndSessionConnection(hostname, walletName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fingerprint := asSessionFingerprint(hostname, walletName)

	token, exists := m.sessionFingerprintToToken[fingerprint]
	if !exists {
		return nil
	}

	m.destroySessionToken(token)

	return nil
}

func (m *Manager) EndAllSessionConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for token := range m.tokenToConnection {
		m.destroySessionToken(token)
	}
}

// ConnectedWallet retrieves the wallet associated to the specified token.
func (m *Manager) ConnectedWallet(hostname string, token Token) (api.ConnectedWallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	connection, exists := m.tokenToConnection[token]
	if !exists {
		return api.ConnectedWallet{}, ErrNoConnectionAssociatedThisToken
	}

	if !connection.policy.HasNoRestrictions() && connection.connectedWallet.Hostname() != hostname {
		return api.ConnectedWallet{}, ErrHostnamesMismatchForThisToken
	}

	now := m.timeService.Now()

	if connection.policy.HasConnectionExpired(now) {
		return api.ConnectedWallet{}, ErrTokenHasExpired
	}

	connection.policy.UpdateActivityDate(now)

	return connection.connectedWallet, nil
}

// ListSessionConnections lists all the session connections as a list of pairs of
// hostname/wallet name.
// The list is sorted, first, by hostname, and, then, by wallet name.
func (m *Manager) ListSessionConnections() []api.Connection {
	m.mu.Lock()
	defer m.mu.Unlock()

	connections := []api.Connection{}
	for _, connection := range m.tokenToConnection {
		if !connection.policy.CanBeEnded() {
			continue
		}

		connections = append(connections, api.Connection{
			Hostname: connection.connectedWallet.Hostname(),
			Wallet:   connection.connectedWallet.Name(),
		})
	}

	sort.SliceStable(connections, func(i, j int) bool {
		if connections[i].Hostname == connections[j].Hostname {
			return connections[i].Wallet < connections[j].Wallet
		}

		return connections[i].Hostname < connections[j].Hostname
	})

	return connections
}

// generateToken generates a new token and ensure it is not already in use to
// avoid collisions.
func (m *Manager) generateToken() Token {
	for {
		token := GenerateToken()
		if _, alreadyExistingToken := m.tokenToConnection[token]; !alreadyExistingToken {
			return token
		}
	}
}

func (m *Manager) destroySessionToken(tokenToDestroy Token) {
	connection, exists := m.tokenToConnection[tokenToDestroy]
	if !exists || !connection.policy.CanBeEnded() {
		return
	}

	walletName := connection.connectedWallet.Name()

	// Remove the session fingerprint associated to the session token.
	for sessionFingerprint, t := range m.sessionFingerprintToToken {
		if t == tokenToDestroy {
			delete(m.sessionFingerprintToToken, sessionFingerprint)
			break
		}
	}

	// Break the link between a token and its associated wallet.
	m.tokenToConnection[tokenToDestroy] = nil
	delete(m.tokenToConnection, tokenToDestroy)

	// Remove the token from the list of token associated to a given wallet.
	tokenIdxGetter := func() int {
		for i, t := range m.walletToTokens[walletName] {
			if tokenToDestroy == t {
				return i
			}
		}
		panic("there is an inconsistent state between the sessions fingerprints and the wallet-to-tokens registry.")
	}

	tokens := m.walletToTokens[walletName]
	tokensCounter := len(tokens)
	tokenIdx := tokenIdxGetter()
	copy(tokens[tokenIdx:], tokens[tokenIdx+1:])
	tokens[tokensCounter-1] = ""
	tokens = tokens[:tokensCounter-1]
	m.walletToTokens[walletName] = tokens
}

func (m *Manager) loadLongLivingConnections() error {
	ctx := context.Background()

	tokenSummaries, err := m.tokenStore.ListTokens()
	if err != nil {
		return err
	}

	for _, tokenSummary := range tokenSummaries {
		tokenDescription, err := m.tokenStore.DescribeToken(tokenSummary.Token)
		if err != nil {
			return fmt.Errorf("could not get information associated to the token %q: %w", tokenDescription.Token.Short(), err)
		}

		// We need to ensure the wallet is unlocked before loading it.
		if err := m.walletStore.UnlockWallet(ctx, tokenDescription.Wallet.Name, tokenDescription.Wallet.Passphrase); err != nil {
			return fmt.Errorf("could not unlock the wallet %q associated to the token %q: %w",
				tokenDescription.Wallet.Name,
				tokenDescription.Token.Short(),
				err)
		}

		w, err := m.walletStore.GetWallet(ctx, tokenDescription.Wallet.Name)
		if err != nil {
			return fmt.Errorf("could not get the information for the wallet %q associated to the token %q: %w",
				tokenDescription.Wallet.Name,
				tokenDescription.Token.Short(),
				err)
		}

		if err := m.loadLongLivingConnection(tokenDescription.Token, w, tokenDescription.ExpirationDate); err != nil {
			return fmt.Errorf("could not initiate the long-living connection associated to the token %q: %w",
				tokenDescription.Token.Short(),
				err)
		}
	}

	return nil
}

func (m *Manager) loadLongLivingConnection(longLivingToken Token, w wallet.Wallet, expiryAt *time.Time) error {
	_, alreadyLoaded := m.walletToTokens[w.Name()]
	if !alreadyLoaded {
		m.walletToTokens[w.Name()] = []Token{}
	}

	m.walletToTokens[w.Name()] = append(m.walletToTokens[w.Name()], longLivingToken)

	cw, err := api.NewLongLivingConnectedWallet(w)
	if err != nil {
		return fmt.Errorf("could not instantiate the connected wallet: %w", err)
	}

	m.tokenToConnection[longLivingToken] = &walletConnection{
		connectedWallet: cw,
		policy: &longLivingConnectionPolicy{
			expirationDate: expiryAt,
		},
	}

	return nil
}

// updateReferenceToWallet is called when the wallet store notices a change in
// the wallets. This way the connection manager is able to reload the connected
// wallets.
func (m *Manager) updateReferenceToWallet(w wallet.Wallet) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokens, isUsed := m.walletToTokens[w.Name()]
	if !isUsed {
		return
	}

	for _, token := range tokens {
		connection := m.tokenToConnection[token]

		cw, err := api.NewConnectedWallet(connection.connectedWallet.Hostname(), w)
		if err != nil {
			// We assume this will work, and there is no reason we end up here.
			continue
		}

		m.tokenToConnection[token] = &walletConnection{
			connectedWallet: cw,
			policy: &sessionPolicy{
				lastActivityDate: m.timeService.Now(),
			},
		}
	}
}

func NewManager(timeService TimeService, walletStore WalletStore, tokenStore TokenStore) (*Manager, error) {
	m := &Manager{
		sessionFingerprintToToken: map[string]Token{},
		tokenToConnection:         map[Token]*walletConnection{},
		walletToTokens:            map[string][]Token{},
		timeService:               timeService,
		walletStore:               walletStore,
		tokenStore:                tokenStore,
	}

	walletStore.OnUpdate(m.updateReferenceToWallet)

	if err := m.loadLongLivingConnections(); err != nil {
		return nil, fmt.Errorf("could not load the long-living connections: %w", err)
	}

	return m, nil
}

func asSessionFingerprint(hostname string, walletName string) string {
	return vgcrypto.HashStrToHex(hostname + "::" + walletName)
}
