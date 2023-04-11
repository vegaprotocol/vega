package connections

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
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

	// timeService is used to resolve the current time to update the last activity
	// time on the token, and figure out their expiration.
	timeService TimeService

	walletStore  WalletStore
	sessionStore SessionStore
	tokenStore   TokenStore

	interactor api.Interactor

	mu sync.Mutex
}

type walletConnection struct {
	// connectedWallet is the projection of the wallet through the permissions
	// and authentication system. On a regular wallet, there are no restrictions
	// on what we can call, which doesn't fit the model of having allowed
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

	now := m.timeService.Now()
	m.tokenToConnection[newToken] = &walletConnection{
		connectedWallet: cw,
		policy: &sessionPolicy{
			expiryDate: now.Add(1 * time.Hour),
		},
	}

	// We ignore this error as tracking the session a nice-to-have feature to
	// ease reconnection after a software reboot. We don't want to prevent the
	// connection because an error on that layer.
	_ = m.sessionStore.TrackSession(Session{
		Token:    newToken,
		Hostname: hostname,
		Wallet:   w.Name(),
	})

	return newToken, nil
}

func (m *Manager) EndSessionConnectionWithToken(token Token) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.destroySessionToken(token)
}

func (m *Manager) EndSessionConnection(hostname, walletName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fingerprint := asSessionFingerprint(hostname, walletName)

	token, exists := m.sessionFingerprintToToken[fingerprint]
	if !exists {
		return
	}

	m.destroySessionToken(token)
}

func (m *Manager) EndAllSessionConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for token := range m.tokenToConnection {
		m.destroySessionToken(token)
	}
}

// ConnectedWallet retrieves the wallet associated to the specified token.
func (m *Manager) ConnectedWallet(ctx context.Context, hostname string, token Token) (api.ConnectedWallet, *jsonrpc.ErrorDetails) {
	m.mu.Lock()
	defer m.mu.Unlock()

	connection, exists := m.tokenToConnection[token]
	if !exists {
		return api.ConnectedWallet{}, serverErrorAuthenticationFailure(ErrNoConnectionAssociatedThisAuthenticationToken)
	}

	now := m.timeService.Now()

	hasExpired := connection.policy.HasConnectionExpired(now)

	if connection.policy.IsLongLivingConnection() {
		if hasExpired {
			return api.ConnectedWallet{}, serverErrorAuthenticationFailure(ErrTokenHasExpired)
		}
	} else {
		traceID := jsonrpc.TraceIDFromContext(ctx)

		if connection.connectedWallet.Hostname() != hostname {
			return api.ConnectedWallet{}, serverErrorAuthenticationFailure(ErrHostnamesMismatchForThisToken)
		}

		isClosed := connection.policy.IsClosed()

		if hasExpired || isClosed {
			if err := m.interactor.NotifyInteractionSessionBegan(ctx, traceID, api.WalletUnlockingWorkflow, 2); err != nil {
				return api.ConnectedWallet{}, api.RequestNotPermittedError(err)
			}
			defer m.interactor.NotifyInteractionSessionEnded(ctx, traceID)

			for {
				if ctx.Err() != nil {
					m.interactor.NotifyError(ctx, traceID, api.ApplicationErrorType, api.ErrRequestInterrupted)
					return api.ConnectedWallet{}, api.RequestInterruptedError(api.ErrRequestInterrupted)
				}

				unlockingReason := fmt.Sprintf("The third-party application %q is attempting access to the locked wallet %q. To unlock this wallet and allow access to all connected apps associated to it, enter its passphrase.", hostname, connection.connectedWallet.Name())

				walletPassphrase, err := m.interactor.RequestPassphrase(ctx, traceID, 1, connection.connectedWallet.Name(), unlockingReason)
				if err != nil {
					if errDetails := api.HandleRequestFlowError(ctx, traceID, m.interactor, err); errDetails != nil {
						return api.ConnectedWallet{}, errDetails
					}
					m.interactor.NotifyError(ctx, traceID, api.InternalErrorType, fmt.Errorf("requesting the wallet passphrase failed: %w", err))
					return api.ConnectedWallet{}, api.InternalError(api.ErrCouldNotConnectToWallet)
				}

				if err := m.walletStore.UnlockWallet(ctx, connection.connectedWallet.Name(), walletPassphrase); err != nil {
					if errors.Is(err, wallet.ErrWrongPassphrase) {
						m.interactor.NotifyError(ctx, traceID, api.UserErrorType, wallet.ErrWrongPassphrase)
						continue
					}
					if errDetails := api.HandleRequestFlowError(ctx, traceID, m.interactor, err); errDetails != nil {
						return api.ConnectedWallet{}, errDetails
					}
					m.interactor.NotifyError(ctx, traceID, api.InternalErrorType, fmt.Errorf("could not unlock the wallet: %w", err))
					return api.ConnectedWallet{}, api.InternalError(api.ErrCouldNotConnectToWallet)
				}
				break
			}

			w, err := m.walletStore.GetWallet(ctx, connection.connectedWallet.Name())
			if err != nil {
				m.interactor.NotifyError(ctx, traceID, api.InternalErrorType, fmt.Errorf("could not retrieve the wallet: %w", err))
				return api.ConnectedWallet{}, api.InternalError(api.ErrCouldNotConnectToWallet)
			}

			// Update the connected wallet for all connections referencing the
			// newly unlocked wallet.
			for _, otherConnection := range m.tokenToConnection {
				if w.Name() != otherConnection.connectedWallet.Name() {
					continue
				}

				cw, err := api.NewConnectedWallet(otherConnection.connectedWallet.Hostname(), w)
				if err != nil {
					m.interactor.NotifyError(ctx, traceID, api.InternalErrorType, fmt.Errorf("could not instantiate the connected wallet for a session connection: %w", err))
					return api.ConnectedWallet{}, api.InternalError(api.ErrCouldNotConnectToWallet)
				}

				otherConnection.connectedWallet = cw
				otherConnection.policy = &sessionPolicy{
					expiryDate: now.Add(1 * time.Hour),
					closed:     false,
				}
			}

			m.interactor.NotifySuccessfulRequest(ctx, traceID, 2, fmt.Sprintf("The wallet %q has been successfully unlocked.", w.Name()))
		}
	}

	return connection.connectedWallet, nil
}

// ListSessionConnections lists all the session connections as a list of pairs of
// hostname/wallet name.
// The list is sorted, first, by hostname, and, then, by wallet name.
func (m *Manager) ListSessionConnections() []api.Connection {
	m.mu.Lock()
	defer m.mu.Unlock()

	connections := make([]api.Connection, 0, len(m.tokenToConnection))
	connectionsCount := 0
	for _, connection := range m.tokenToConnection {
		if connection.policy.IsLongLivingConnection() {
			continue
		}

		connections = append(connections, api.Connection{
			Hostname: connection.connectedWallet.Hostname(),
			Wallet:   connection.connectedWallet.Name(),
		})

		connectionsCount++
	}
	connections = connections[0:connectionsCount]

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
	if !exists || connection.policy.IsLongLivingConnection() {
		return
	}

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
}

func (m *Manager) loadLongLivingConnections(ctx context.Context) error {
	tokenSummaries, err := m.tokenStore.ListTokens()
	if err != nil {
		return err
	}

	for _, tokenSummary := range tokenSummaries {
		tokenDescription, err := m.tokenStore.DescribeToken(tokenSummary.Token)
		if err != nil {
			return fmt.Errorf("could not get information associated to the token %q: %w", tokenDescription.Token.Short(), err)
		}

		if err := m.loadLongLivingConnection(ctx, tokenDescription); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) loadLongLivingConnection(ctx context.Context, tokenDescription TokenDescription) error {
	if err := m.walletStore.UnlockWallet(ctx, tokenDescription.Wallet.Name, tokenDescription.Wallet.Passphrase); err != nil {
		// We don't properly handle wallets renaming, nor wallets passphrase
		// update in the token file automatically. We only support a direct
		// update of the token file.
		return fmt.Errorf("could not unlock the wallet %q associated to the token %q: %w",
			tokenDescription.Wallet.Name,
			tokenDescription.Token.Short(),
			err)
	}

	w, err := m.walletStore.GetWallet(ctx, tokenDescription.Wallet.Name)
	if err != nil {
		// This should not happen because we just unlocked the wallet.
		return fmt.Errorf("could not get the information for the wallet %q associated to the token %q: %w",
			tokenDescription.Wallet.Name,
			tokenDescription.Token.Short(),
			err)
	}

	m.tokenToConnection[tokenDescription.Token] = &walletConnection{
		connectedWallet: api.NewLongLivingConnectedWallet(w),
		policy: &longLivingConnectionPolicy{
			expirationDate: tokenDescription.ExpirationDate,
		},
	}

	return nil
}

func (m *Manager) loadPreviousSessionConnections(ctx context.Context) error {
	sessions, err := m.sessionStore.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("could not list the sessions: %w", err)
	}

	now := m.timeService.Now()

	for _, session := range sessions {
		sessionFingerprint := asSessionFingerprint(session.Hostname, session.Wallet)
		m.sessionFingerprintToToken[sessionFingerprint] = session.Token

		// If the wallet in the session store doesn't exist, we destroy that session
		// and move onto the next.
		walletExists, err := m.walletStore.WalletExists(ctx, session.Wallet)
		if err != nil {
			return fmt.Errorf("could not verify if the wallet %q exists: %w", session.Wallet, err)
		}

		if !walletExists {
			err := m.sessionStore.DeleteSession(ctx, session.Token)
			if err != nil {
				return fmt.Errorf("could not delete the session with token %q: %w", session.Token, err)
			}
			continue
		}

		// If the wallet is already unlocked, we fully restore the connection.
		isAlreadyUnlocked, err := m.walletStore.IsWalletAlreadyUnlocked(ctx, session.Wallet)
		if err != nil {
			return fmt.Errorf("could not verify wether the wallet %q is locked or not: %w", session.Wallet, err)
		}

		var connectedWallet api.ConnectedWallet
		isClosed := true
		if isAlreadyUnlocked {
			w, err := m.walletStore.GetWallet(ctx, session.Wallet)
			if err != nil {
				return fmt.Errorf("could not retrieve the wallet %q: %w", session.Wallet, err)
			}
			cw, err := api.NewConnectedWallet(session.Hostname, w)
			if err != nil {
				return fmt.Errorf("could not instantiate the connected wallet for a session connection: %w", err)
			}
			connectedWallet = cw
			isClosed = false
		} else {
			connectedWallet = api.NewDisconnectedWallet(session.Hostname, session.Wallet)
		}

		m.tokenToConnection[session.Token] = &walletConnection{
			// We are unable to build the connected the wallet at this point.
			// We will have one when the user explicitly
			connectedWallet: connectedWallet,
			policy: &sessionPolicy{
				// Since this session is being reloaded, we consider it to be
				// expired.
				expiryDate: now,
				closed:     isClosed,
			},
		}
	}

	return nil
}

// refreshConnections is called when the wallet store notices a change in
// the wallets. This way the connection manager is able to reload the connected
// wallets.
func (m *Manager) refreshConnections(_ context.Context, event wallet.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch event.Type {
	case wallet.WalletRemovedEventType:
		m.destroyConnectionsUsingThisWallet(event.Data.(wallet.WalletRemovedEventData).Name)
	case wallet.UnlockedWalletUpdatedEventType:
		m.updateConnectionsUsingThisWallet(event.Data.(wallet.UnlockedWalletUpdatedEventData).UpdatedWallet)
	case wallet.WalletHasBeenLockedEventType:
		m.closeConnectionsUsingThisWallet(event.Data.(wallet.WalletHasBeenLockedEventData).Name)
	case wallet.WalletRenamedEventType:
		data := event.Data.(wallet.WalletRenamedEventData)
		m.updateConnectionsUsingThisRenamedWallet(data.PreviousName, data.NewName)
	}
}

// destroyConnectionUsingThisWallet close the connection, dereference it, and
// remove it from the session store.
func (m *Manager) destroyConnectionsUsingThisWallet(walletName string) {
	ctx := context.Background()

	for token, connection := range m.tokenToConnection {
		if connection.connectedWallet.Name() != walletName {
			continue
		}

		connection.policy.SetAsForcefullyClose()

		delete(m.tokenToConnection, token)

		// We ignore the error in a best-effort to have the session store clean
		// up.
		_ = m.sessionStore.DeleteSession(ctx, token)
	}
}

func (m *Manager) updateConnectionsUsingThisWallet(w wallet.Wallet) {
	for token, connection := range m.tokenToConnection {
		if connection.connectedWallet.Name() != w.Name() {
			continue
		}

		var updatedConnectedWallet api.ConnectedWallet
		if connection.policy.IsLongLivingConnection() {
			updatedConnectedWallet = api.NewLongLivingConnectedWallet(w)
		} else {
			updatedConnectedWallet, _ = api.NewConnectedWallet(connection.connectedWallet.Hostname(), w)
		}

		m.tokenToConnection[token].connectedWallet = updatedConnectedWallet
	}
}

func (m *Manager) refreshLongLivingTokens(ctx context.Context, activeTokensDescriptions ...TokenDescription) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// We need to find the new tokens among the active ones, so, we build a
	// registry with all the active tokens. Then, we remove the tracked token
	// when found. We will end up with the new tokens, only.
	activeTokens := make(map[Token]TokenDescription, len(activeTokensDescriptions))
	for _, tokenDescription := range activeTokensDescriptions {
		activeTokens[tokenDescription.Token] = tokenDescription
	}

	isActiveToken := func(token Token) (TokenDescription, bool) {
		activeTokenDescription, isTracked := activeTokens[token]
		if isTracked {
			delete(activeTokens, token)
		}
		return activeTokenDescription, isTracked
	}

	// First, we update of the tokens we already track.
	for token, connection := range m.tokenToConnection {
		if !connection.policy.IsLongLivingConnection() {
			continue
		}

		activeToken, isActive := isActiveToken(token)
		if !isActive {
			// If the token could not be found in the active tokens, this means
			// the token has been deleted from the token store. Thus, we close the
			// connection.
			delete(m.tokenToConnection, token)
			continue
		}

		_ = m.loadLongLivingConnection(ctx, activeToken)
	}

	// Then, we load the new tokens.
	for _, tokenDescription := range activeTokens {
		_ = m.loadLongLivingConnection(ctx, tokenDescription)
	}
}

// closeConnectionsUsingThisWallet defines the connection as closed. It keeps track of it
// but next time it will be used the application will request for the passphrase
// to reinstate it.
func (m *Manager) closeConnectionsUsingThisWallet(walletName string) {
	for _, connection := range m.tokenToConnection {
		if connection.connectedWallet.Name() != walletName {
			continue
		}

		connection.policy.SetAsForcefullyClose()
	}
}

func (m *Manager) updateConnectionsUsingThisRenamedWallet(previousWalletName, newWalletName string) {
	var _updatedWallet wallet.Wallet

	// This acts as a cached getter, to avoid multiple or useless fetch.
	getUpdatedWallet := func() wallet.Wallet {
		if _updatedWallet == nil {
			w, _ := m.walletStore.GetWallet(context.Background(), newWalletName)
			_updatedWallet = w
		}
		return _updatedWallet
	}

	for _, connection := range m.tokenToConnection {
		if connection.connectedWallet.Name() != previousWalletName {
			continue
		}

		if connection.policy.IsLongLivingConnection() {
			connection.connectedWallet = api.NewLongLivingConnectedWallet(
				getUpdatedWallet(),
			)
		}

		connection.connectedWallet, _ = api.NewConnectedWallet(
			connection.connectedWallet.Hostname(),
			getUpdatedWallet(),
		)
	}
}

func NewManager(timeService TimeService, walletStore WalletStore, tokenStore TokenStore, sessionStore SessionStore, interactor api.Interactor) (*Manager, error) {
	m := &Manager{
		tokenToConnection:         map[Token]*walletConnection{},
		sessionFingerprintToToken: map[string]Token{},
		timeService:               timeService,
		walletStore:               walletStore,
		sessionStore:              sessionStore,
		tokenStore:                tokenStore,
		interactor:                interactor,
	}

	walletStore.OnUpdate(m.refreshConnections)
	tokenStore.OnUpdate(m.refreshLongLivingTokens)

	ctx := context.Background()

	if err := m.loadLongLivingConnections(ctx); err != nil {
		return nil, fmt.Errorf("could not load the long-living connections: %w", err)
	}

	if err := m.loadPreviousSessionConnections(ctx); err != nil {
		return nil, fmt.Errorf("could not load the previous session connections: %w", err)
	}

	return m, nil
}

func asSessionFingerprint(hostname string, walletName string) string {
	return vgcrypto.HashStrToHex(hostname + "::" + walletName)
}

func serverErrorAuthenticationFailure(err error) *jsonrpc.ErrorDetails {
	return jsonrpc.NewServerError(api.ErrorCodeAuthenticationFailure, err)
}
