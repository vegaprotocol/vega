package session

import (
	"errors"
	"fmt"
	"sort"
	"time"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

var (
	ErrNoWalletConnected                       = errors.New("no wallet connected")
	ErrCannotEndLongLivingSessions             = errors.New("sessions attached to long-living tokens cannot be ended")
	ErrGeneratedTokenCollidesWithExistingToken = errors.New("the generated token collides with an existing token")
	ErrAPITokenExpired                         = errors.New("the token has expired")
)

// Sessions holds the live sessions.
type Sessions struct {
	// shortLivingConnectionFingerprints holds the hash of the wallet and the
	// hostname in use in the sessions as key and the token as value. It's used
	// to retrieve the token if the client quit the third-party application
	// without disconnecting the wallet first.
	shortLivingConnectionFingerprints map[string]string

	// connectedWallets holds the wallet resources and information by token.
	connectedWallets map[string]*ConnectedWallet
}

// ConnectWallet initiates a wallet connection and load associated resources in
// it. If a connection already exists, it's disconnected and a new token is
// generated.
func (s *Sessions) ConnectWallet(hostname string, w wallet.Wallet) (string, error) {
	fingerprint := toFingerprint(hostname, w.Name())
	if token, ok := s.shortLivingConnectionFingerprints[fingerprint]; ok {
		// We already have a connection for that wallet and hostname, we destroy
		// it, before creating a new one.
		if err := s.DisconnectWalletWithToken(token); err != nil {
			// This should not happen.
			return "", fmt.Errorf("could not disconnect the wallet before reconnection: %w", err)
		}
	}

	connectedWallet, err := NewConnectedWallet(hostname, w)
	if err != nil {
		return "", fmt.Errorf("could not load the wallet: %w", err)
	}

	token := GenerateToken()

	if _, alreadyExistingToken := s.connectedWallets[token]; alreadyExistingToken {
		return "", ErrGeneratedTokenCollidesWithExistingToken
	}

	s.shortLivingConnectionFingerprints[fingerprint] = token
	s.connectedWallets[token] = connectedWallet

	return token, nil
}

func (s *Sessions) ConnectWalletForLongLivingConnection(
	token string, w wallet.Wallet, now time.Time, expiry *time.Time,
) error {
	connectedWallet, err := NewLongLivingConnectedWallet(w, now, expiry)
	if err != nil {
		return fmt.Errorf("could not load the wallet: %w", err)
	}
	s.connectedWallets[token] = connectedWallet
	return nil
}

// DisconnectWalletWithToken looks for the connected wallet associated to the
// token, unloads its resources, and revokes the token.
// It does not fail. Non-existing token does nothing.
func (s *Sessions) DisconnectWalletWithToken(token string) error {
	connectedWallet, ok := s.connectedWallets[token]
	if !ok {
		return nil
	}
	if connectedWallet.noRestrictions {
		return ErrCannotEndLongLivingSessions
	}

	fingerprint := toFingerprint(connectedWallet.Hostname, connectedWallet.Wallet.Name())
	delete(s.shortLivingConnectionFingerprints, fingerprint)
	delete(s.connectedWallets, token)
	return nil
}

// DisconnectWallet unloads the connected wallet resources and revokes the token.
// It does not fail. Non-existing token does nothing.
// This doesn't work for long-living connections.
func (s *Sessions) DisconnectWallet(hostname, wallet string) {
	fingerprint := toFingerprint(hostname, wallet)
	token := s.shortLivingConnectionFingerprints[fingerprint]
	delete(s.shortLivingConnectionFingerprints, fingerprint)
	delete(s.connectedWallets, token)
}

func (s *Sessions) DisconnectAllWallets() {
	// Long-living connections should be kept alive.
	longLivingConnections := map[string]*ConnectedWallet{}
	for token, connectedWallet := range s.connectedWallets {
		if !connectedWallet.RequireInteraction() {
			longLivingConnections[token] = connectedWallet
		}
	}

	s.connectedWallets = longLivingConnections
	s.shortLivingConnectionFingerprints = map[string]string{}
}

// GetConnectedWallet retrieves the resources and information of the
// connected wallet, associated to the specified token.
func (s *Sessions) GetConnectedWallet(token string, now time.Time) (*ConnectedWallet, error) {
	connectedWallet, ok := s.connectedWallets[token]
	if !ok {
		return nil, ErrNoWalletConnected
	}

	if err := connectedWallet.Expired(now); err != nil {
		return nil, ErrAPITokenExpired
	}

	return connectedWallet, nil
}

// ListConnections lists all the live connections as a list of pairs of
// hostname/wallet name.
// The list is sorted, first, by hostname, and, then, by wallet name.
func (s *Sessions) ListConnections() []Connection {
	connections := make([]Connection, 0, len(s.connectedWallets))
	for _, connectedWallet := range s.connectedWallets {
		connections = append(connections, Connection{
			Hostname: connectedWallet.Hostname,
			Wallet:   connectedWallet.Wallet.Name(),
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

func NewSessions() *Sessions {
	return &Sessions{
		shortLivingConnectionFingerprints: map[string]string{},
		connectedWallets:                  map[string]*ConnectedWallet{},
	}
}

type Connection struct {
	// Hostname is the hostname for which the connection is set.
	Hostname string `json:"hostname"`
	// Wallet is the wallet selected by the client for this connection.
	Wallet string `json:"wallet"`
}

// ConnectedWallet contains the resources and the information of the current
// connection, required by the wallet handlers to work, and based on the
// permissions the client has set.
type ConnectedWallet struct {
	// Hostname is the hostname for which the connection is set.
	Hostname string
	// Wallet is the wallet selected by the client for this connection.
	Wallet wallet.Wallet
	// RestrictedKeys holds the keys that have been selected by the client
	// during the permissions request.
	RestrictedKeys map[string]wallet.KeyPair

	// An optional expiry date for this token
	Expiry *time.Time

	// noRestrictions is a hack to know if we should skip permission
	// verification when we are connected with a long-living API token.
	noRestrictions bool
}

func (s *ConnectedWallet) Expired(now time.Time) error {
	if s.Expiry != nil && s.Expiry.Before(now) {
		return ErrAPITokenExpired
	}
	return nil
}

// RequireInteraction tells if an interaction with the user is needed for
// supervision is required or not.
// It is related to the type of API token that is used for this connection.
// If it's a long-living token, then no interaction is required.
func (s *ConnectedWallet) RequireInteraction() bool {
	return !s.noRestrictions
}

func (s *ConnectedWallet) Permissions() wallet.Permissions {
	if s.noRestrictions {
		return wallet.PermissionsWithoutRestrictions()
	}
	return s.Wallet.Permissions(s.Hostname)
}

// CanUseKey determines is the permissions allow the specified key to be used,
// and ensure the key exist and is not tainted.
func (s *ConnectedWallet) CanUseKey(pubKey string) bool {
	if !s.Permissions().CanUseKey(pubKey) {
		return false
	}

	kp, err := s.Wallet.DescribeKeyPair(pubKey)
	if err != nil {
		return false
	}

	return !kp.IsTainted()
}

func (s *ConnectedWallet) UpdatePermissions(perms wallet.Permissions) error {
	if err := s.Wallet.UpdatePermissions(s.Hostname, perms); err != nil {
		return fmt.Errorf("could not update permission on the wallet: %w", err)
	}

	// Since we just updated the permissions on the wallet, we need to reload
	// the restricted keys to match the update.
	if err := s.loadRestrictedKeys(); err != nil {
		return err
	}

	return nil
}

func (s *ConnectedWallet) ReloadWithWallet(updatedWallet wallet.Wallet) error {
	s.Wallet = updatedWallet

	if err := s.loadRestrictedKeys(); err != nil {
		return err
	}

	return nil
}

func (s *ConnectedWallet) loadRestrictedKeys() error {
	perms := s.Permissions()

	if !perms.PublicKeys.Enabled() {
		return nil
	}

	if perms.PublicKeys.HasRestrictedKeys() {
		for _, pubKey := range perms.PublicKeys.RestrictedKeys {
			keyPair, err := s.Wallet.DescribeKeyPair(pubKey)
			if err != nil {
				return fmt.Errorf("could not load the key pair associated to the public key %q: %w", pubKey, err)
			}
			s.RestrictedKeys[keyPair.PublicKey()] = keyPair
		}
		return nil
	}

	// If there is no restricted keys set, we load all valid keys.
	for _, keyPair := range s.Wallet.ListKeyPairs() {
		if !keyPair.IsTainted() {
			s.RestrictedKeys[keyPair.PublicKey()] = keyPair
		}
	}

	return nil
}

func NewConnectedWallet(hostname string, w wallet.Wallet) (*ConnectedWallet, error) {
	s := &ConnectedWallet{
		noRestrictions: false,
		Hostname:       hostname,
		Wallet:         w,
		RestrictedKeys: map[string]wallet.KeyPair{},
	}

	if err := s.loadRestrictedKeys(); err != nil {
		return nil, fmt.Errorf("could not load the restricted keys: %w", err)
	}

	return s, nil
}

func NewLongLivingConnectedWallet(w wallet.Wallet, now time.Time, expiry *time.Time) (*ConnectedWallet, error) {
	s := &ConnectedWallet{
		noRestrictions: true,
		Hostname:       "",
		Wallet:         w,
		Expiry:         expiry,
		RestrictedKeys: map[string]wallet.KeyPair{},
	}
	if err := s.Expired(now); err != nil {
		return nil, err
	}

	if err := s.loadRestrictedKeys(); err != nil {
		return nil, fmt.Errorf("could not load the restricted keys: %w", err)
	}

	return s, nil
}

func toFingerprint(hostname string, walletName string) string {
	return vgcrypto.HashStrToHex(hostname + "::" + walletName)
}
