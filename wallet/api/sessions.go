package api

import (
	"errors"
	"fmt"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

var ErrNoWalletConnected = errors.New("no wallet connected")

type Sessions struct {
	// fingerprints holds the hash of the wallet and the hostname in use in the
	// sessions as key and the token as value. It's used to retrieve the token
	// if the client quit the third-party application without disconnecting the
	// wallet first.
	fingerprints map[string]string
	// connectedWallets holds the wallet resources and information by token.
	connectedWallets map[string]*ConnectedWallet
}

// ConnectWallet initiates a wallet connection and load associated resources in
// it. If a connection already exists, it's disconnected and a new token is
// generated.
func (s *Sessions) ConnectWallet(hostname string, w wallet.Wallet) (string, error) {
	fingerprint := toFingerprint(hostname, w)
	if token, ok := s.fingerprints[fingerprint]; ok {
		// We already have a connection for that wallet and hostname, we destroy
		// it.
		s.DisconnectWallet(token)
	}

	connectedWallet, err := NewConnectedWallet(hostname, w)
	if err != nil {
		return "", fmt.Errorf("could not load the wallet: %w", err)
	}

	token := vgrand.RandomStr(64)

	s.fingerprints[fingerprint] = token
	s.connectedWallets[token] = connectedWallet

	return token, nil
}

// DisconnectWallet unloads the connected wallet resources and revokes the token.
// It does not fail. Non-existing token does nothing.
func (s *Sessions) DisconnectWallet(token string) {
	connectedWallet, ok := s.connectedWallets[token]
	if !ok {
		return
	}

	fingerprint := toFingerprint(connectedWallet.Hostname, connectedWallet.Wallet)
	delete(s.fingerprints, fingerprint)
	delete(s.connectedWallets, token)
}

// GetConnectedWallet retrieves the resources and information of the
// connected wallet, associated to the specified token.
func (s *Sessions) GetConnectedWallet(token string) (*ConnectedWallet, error) {
	connectedWallet, ok := s.connectedWallets[token]
	if !ok {
		return nil, ErrNoWalletConnected
	}

	return connectedWallet, nil
}

func NewSessions() *Sessions {
	return &Sessions{
		fingerprints:     map[string]string{},
		connectedWallets: map[string]*ConnectedWallet{},
	}
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
}

func (s *ConnectedWallet) Permissions() wallet.Permissions {
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
		Hostname:       hostname,
		Wallet:         w,
		RestrictedKeys: map[string]wallet.KeyPair{},
	}

	if err := s.loadRestrictedKeys(); err != nil {
		return nil, fmt.Errorf("could not load the restricted keys: %w", err)
	}

	return s, nil
}

func toFingerprint(hostname string, w wallet.Wallet) string {
	return vgcrypto.HashStrToHex(hostname + "::" + w.Name())
}
