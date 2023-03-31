package api

import (
	"fmt"

	"code.vegaprotocol.io/vega/wallet/wallet"
)

// ConnectedWallet is the projection of the wallet through the permissions
// and authentication system. On a regular wallet, there are no restrictions
// on what we can call, which doesn't fit the model of having allowed
// access, so we wrap the "regular wallet" behind the "connected wallet".
type ConnectedWallet struct {
	// name is the name of the wallet.
	name string

	// Hostname is the hostname for which the connection is set.
	hostname string

	// allowedKeys holds the keys that have been selected by the client
	// during the permissions request.
	// The order should match the order of generation in the wallet.
	allowedKeys []AllowedKey

	// noRestrictions is a hack to know if we should skip permission
	// verification when we are connected with a long-living API token.
	noRestrictions bool

	canListKeys bool
}

func (s *ConnectedWallet) Name() string {
	return s.name
}

// Hostname returns the hostname for which the connection has been set.
// For long-living connections, the hostname is empty  as there is no
// restrictions for that type of connection.
func (s *ConnectedWallet) Hostname() string {
	return s.hostname
}

// AllowedKeys returns the keys a connection has access to. If a third-party
// application tries to use a keys that does not belong to this set, then the
// request should fail.
func (s *ConnectedWallet) AllowedKeys() []AllowedKey {
	return s.allowedKeys
}

// RequireInteraction tells if an interaction with the user is needed for
// supervision is required or not.
// It is related to the type of API token that is used for this connection.
// If it's a long-living token, then no interaction is required.
func (s *ConnectedWallet) RequireInteraction() bool {
	return !s.noRestrictions
}

func (s *ConnectedWallet) CanListKeys() bool {
	if s.noRestrictions {
		return true
	}
	return s.canListKeys
}

// CanUseKey determines if the permissions allow the specified key to be used.
func (s *ConnectedWallet) CanUseKey(publicKeyToUse string) bool {
	for _, allowedKey := range s.allowedKeys {
		if allowedKey.PublicKey() == publicKeyToUse {
			return true
		}
	}

	return false
}

func (s *ConnectedWallet) RefreshFromWallet(freshWallet wallet.Wallet) error {
	if s.noRestrictions {
		s.allowedKeys = allUsableKeys(freshWallet)
		return nil
	}

	rks, err := allowedKeys(freshWallet, s.hostname)
	if err != nil {
		return fmt.Errorf("could not resolve the allowed keys when refreshing the connection: %w", err)
	}

	s.canListKeys = rks != nil
	s.allowedKeys = rks

	return nil
}

type AllowedKey struct {
	publicKey string
	name      string
}

func (r AllowedKey) PublicKey() string {
	return r.publicKey
}

func (r AllowedKey) Name() string {
	return r.name
}

func NewConnectedWallet(hostname string, w wallet.Wallet) (ConnectedWallet, error) {
	rks, err := allowedKeys(w, hostname)
	if err != nil {
		return ConnectedWallet{}, fmt.Errorf("could not resolve the allowed keys: %w", err)
	}

	return ConnectedWallet{
		noRestrictions: false,
		canListKeys:    rks != nil,
		allowedKeys:    rks,
		hostname:       hostname,
		name:           w.Name(),
	}, nil
}

func NewLongLivingConnectedWallet(w wallet.Wallet) ConnectedWallet {
	return ConnectedWallet{
		noRestrictions: true,
		canListKeys:    true,
		allowedKeys:    allUsableKeys(w),
		hostname:       "",
		name:           w.Name(),
	}
}

func allowedKeys(w wallet.Wallet, hostname string) ([]AllowedKey, error) {
	perms := w.Permissions(hostname)

	if !perms.PublicKeys.Enabled() {
		return nil, nil
	}

	if !perms.PublicKeys.HasAllowedKeys() {
		// If there is no allowed keys set for this hostname, we load all valid
		// keys.
		return allUsableKeys(w), nil
	}

	allowedKeys := make([]AllowedKey, 0, len(perms.PublicKeys.AllowedKeys))
	for _, pubKey := range perms.PublicKeys.AllowedKeys {
		keyPair, err := w.DescribeKeyPair(pubKey)
		if err != nil {
			return nil, fmt.Errorf("could not load the key pair associated to the public key %q: %w", pubKey, err)
		}
		// There is no need to check for the tainted keys, here, as this list
		// should only contain usable keys.
		allowedKeys = append(allowedKeys, AllowedKey{
			publicKey: keyPair.PublicKey(),
			name:      keyPair.Name(),
		})
	}
	return allowedKeys, nil
}

func allUsableKeys(w wallet.Wallet) []AllowedKey {
	allKeyPairs := w.ListKeyPairs()
	allowedKeys := make([]AllowedKey, 0, len(allKeyPairs))
	for _, keyPair := range allKeyPairs {
		if !keyPair.IsTainted() {
			allowedKeys = append(allowedKeys, AllowedKey{
				publicKey: keyPair.PublicKey(),
				name:      keyPair.Name(),
			})
		}
	}
	return allowedKeys
}
