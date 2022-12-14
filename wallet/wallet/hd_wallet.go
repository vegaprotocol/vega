package wallet

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/tyler-smith/go-bip39"
	"github.com/vegaprotocol/go-slip10"
)

const (
	// MaxEntropyByteSize is the entropy bytes size used for recovery phrase
	// generation.
	MaxEntropyByteSize = 256
	// MagicIndex is the registered HD wallet index for Vega's wallets.
	MagicIndex = 1789
	// OriginIndex is a constant index used to derive a node from the master
	// node. The resulting node will be used to generate the cryptographic keys.
	OriginIndex = slip10.FirstHardenedIndex + MagicIndex
)

var ErrCannotSetRestrictedKeysWithNoAccess = errors.New("can't set restricted keys with \"none\" access")

type HDWallet struct {
	keyDerivationVersion uint32
	name                 string
	keyRing              *HDKeyRing

	// node is the node from which the cryptographic keys are generated. This is
	// not the master node. This is a node derived from the master. Its
	// derivation index is constant (see OriginIndex). This node is referred as
	// "wallet node".
	node        *slip10.Node
	id          string
	permissions map[string]Permissions
}

// NewHDWallet creates a wallet with auto-generated recovery phrase. This is
// useful to create a brand-new wallet, without having to take care of the
// recovery phrase generation.
// The generated recovery phrase is returned alongside the created wallet.
func NewHDWallet(name string) (*HDWallet, string, error) {
	recoveryPhrase, err := NewRecoveryPhrase()
	if err != nil {
		return nil, "", err
	}

	w, err := ImportHDWallet(name, recoveryPhrase, LatestVersion)
	if err != nil {
		return nil, "", err
	}

	return w, recoveryPhrase, err
}

// ImportHDWallet creates a wallet based on the recovery phrase in input. This
// is useful import or retrieve a wallet.
func ImportHDWallet(name, recoveryPhrase string, keyDerivationVersion uint32) (*HDWallet, error) {
	recoveryPhrase = sanitizeRecoveryPhrase(recoveryPhrase)

	if !bip39.IsMnemonicValid(recoveryPhrase) {
		return nil, ErrInvalidRecoveryPhrase
	}

	if !IsKeyDerivationVersionSupported(keyDerivationVersion) {
		return nil, NewUnsupportedWalletVersionError(keyDerivationVersion)
	}

	walletNode, err := deriveWalletNodeFromRecoveryPhrase(recoveryPhrase)
	if err != nil {
		return nil, err
	}

	return &HDWallet{
		keyDerivationVersion: keyDerivationVersion,
		name:                 name,
		node:                 walletNode,
		id:                   walletID(walletNode),
		keyRing:              NewHDKeyRing(),
		permissions:          map[string]Permissions{},
	}, nil
}

func (w *HDWallet) KeyDerivationVersion() uint32 {
	return w.keyDerivationVersion
}

func (w *HDWallet) Name() string {
	return w.name
}

func (w *HDWallet) ID() string {
	return w.id
}

func (w *HDWallet) Type() string {
	if w.IsIsolated() {
		return "HD wallet (isolated)"
	}
	return "HD wallet"
}

func (w *HDWallet) SetName(newName string) {
	w.name = newName
}

func (w *HDWallet) HasPublicKey(pubKey string) bool {
	_, exists := w.keyRing.FindPair(pubKey)
	return exists
}

// DescribeKeyPair returns all the information associated with a public key.
func (w *HDWallet) DescribeKeyPair(pubKey string) (KeyPair, error) {
	keyPair, ok := w.keyRing.FindPair(pubKey)
	if !ok {
		return nil, ErrPubKeyDoesNotExist
	}
	return &keyPair, nil
}

// MasterKey returns all the information associated to a master key pair.
func (w *HDWallet) MasterKey() (MasterKeyPair, error) {
	if w.IsIsolated() {
		return nil, ErrIsolatedWalletDoesNotHaveMasterKey
	}

	pubKey, priKey := w.node.Keypair()
	keyPair, err := NewHDMasterKeyPair(pubKey, priKey)
	if err != nil {
		return nil, err
	}

	return keyPair, nil
}

// DescribePublicKey returns all the information associated to a public key,
// except the private key.
func (w *HDWallet) DescribePublicKey(pubKey string) (PublicKey, error) {
	keyPair, ok := w.keyRing.FindPair(pubKey)
	if !ok {
		return nil, ErrPubKeyDoesNotExist
	}

	publicKey := keyPair.ToPublicKey()
	return &publicKey, nil
}

// ListPublicKeys lists the public keys with their information. The private keys
// are not returned.
func (w *HDWallet) ListPublicKeys() []PublicKey {
	originalKeys := w.keyRing.ListKeyPairs()
	keys := make([]PublicKey, len(originalKeys))
	for i, key := range originalKeys {
		publicKey := key.ToPublicKey()
		keys[i] = &publicKey
	}
	return keys
}

// ListKeyPairs lists the key pairs. Be careful, it contains the private key.
func (w *HDWallet) ListKeyPairs() []KeyPair {
	originalKeys := w.keyRing.ListKeyPairs()
	keys := make([]KeyPair, len(originalKeys))
	for i, key := range originalKeys {
		keys[i] = key.DeepCopy()
	}
	return keys
}

// GenerateKeyPair generates a new key pair from a node, that is derived from
// the wallet node.
func (w *HDWallet) GenerateKeyPair(meta []Metadata) (KeyPair, error) {
	if w.IsIsolated() {
		return nil, ErrIsolatedWalletCantGenerateKeys
	}
	nextIndex := w.keyRing.NextIndex()

	keyNode, err := w.deriveKeyNode(nextIndex)
	if err != nil {
		return nil, err
	}

	publicKey, privateKey := keyNode.Keypair()
	keyPair, err := NewHDKeyPair(nextIndex, publicKey, privateKey)
	if err != nil {
		return nil, err
	}

	_ = keyPair.UpdateMetadata(meta)

	w.keyRing.Upsert(*keyPair)

	return keyPair.DeepCopy(), nil
}

// TaintKey marks a key as tainted.
func (w *HDWallet) TaintKey(pubKey string) error {
	keyPair, ok := w.keyRing.FindPair(pubKey)
	if !ok {
		return ErrPubKeyDoesNotExist
	}

	if err := keyPair.Taint(); err != nil {
		return err
	}

	w.keyRing.Upsert(keyPair)

	w.removeTaintedKeyFromRestrictedKeys(keyPair)

	return nil
}

// UntaintKey remove the taint on a key.
func (w *HDWallet) UntaintKey(pubKey string) error {
	keyPair, ok := w.keyRing.FindPair(pubKey)
	if !ok {
		return ErrPubKeyDoesNotExist
	}

	if err := keyPair.Untaint(); err != nil {
		return err
	}

	w.keyRing.Upsert(keyPair)

	return nil
}

// AnnotateKey replaces the key's metadata by the new ones.
// If the `name` metadata is missing it's added automatically with a default.
func (w *HDWallet) AnnotateKey(pubKey string, meta []Metadata) ([]Metadata, error) {
	keyPair, ok := w.keyRing.FindPair(pubKey)
	if !ok {
		return nil, ErrPubKeyDoesNotExist
	}

	updatedMeta := keyPair.UpdateMetadata(meta)

	w.keyRing.Upsert(keyPair)

	return updatedMeta, nil
}

func (w *HDWallet) SignAny(pubKey string, data []byte) ([]byte, error) {
	keyPair, ok := w.keyRing.FindPair(pubKey)
	if !ok {
		return nil, ErrPubKeyDoesNotExist
	}

	return keyPair.SignAny(data)
}

func (w *HDWallet) VerifyAny(pubKey string, data, sig []byte) (bool, error) {
	keyPair, ok := w.keyRing.FindPair(pubKey)
	if !ok {
		return false, ErrPubKeyDoesNotExist
	}

	return keyPair.VerifyAny(data, sig)
}

func (w *HDWallet) SignTx(pubKey string, data []byte) (*Signature, error) {
	keyPair, ok := w.keyRing.FindPair(pubKey)
	if !ok {
		return nil, ErrPubKeyDoesNotExist
	}

	return keyPair.Sign(data)
}

func (w *HDWallet) IsolateWithKey(pubKey string) (Wallet, error) {
	keyPair, ok := w.keyRing.FindPair(pubKey)
	if !ok {
		return nil, ErrPubKeyDoesNotExist
	}

	if keyPair.IsTainted() {
		return nil, ErrPubKeyIsTainted
	}

	return &HDWallet{
		keyDerivationVersion: w.keyDerivationVersion,
		name:                 fmt.Sprintf("%s.%s.isolated", w.name, keyPair.PublicKey()[0:8]),
		keyRing:              LoadHDKeyRing([]HDKeyPair{keyPair}),
		id:                   w.id,
		permissions:          w.permissions,
	}, nil
}

func (w *HDWallet) IsIsolated() bool {
	return w.node == nil
}

func (w *HDWallet) Permissions(hostname string) Permissions {
	perms, ok := w.permissions[hostname]
	if !ok {
		return DefaultPermissions()
	}
	return perms
}

func (w *HDWallet) PermittedHostnames() []string {
	hostnames := make([]string, 0, len(w.permissions))
	for hostname := range w.permissions {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)
	return hostnames
}

func (w *HDWallet) RevokePermissions(hostname string) {
	delete(w.permissions, hostname)
}

func (w *HDWallet) PurgePermissions() {
	w.permissions = map[string]Permissions{}
}

func (w *HDWallet) UpdatePermissions(hostname string, perms Permissions) error {
	// Set defaults.
	if perms.PublicKeys.Access == "" {
		perms.PublicKeys.Access = NoAccess
	}

	if err := ensurePublicKeysPermissionsConsistency(w, perms); err != nil {
		return fmt.Errorf("inconsistent permissions setup: %w", err)
	}

	w.permissions[hostname] = perms
	return nil
}

func (w *HDWallet) MarshalJSON() ([]byte, error) {
	jsonW := jsonHDWallet{
		KeyDerivationVersion: w.KeyDerivationVersion(),
		Node:                 w.node,
		ID:                   w.id,
		Keys:                 w.keyRing.ListKeyPairs(),
		Permissions:          w.permissions,
	}

	if jsonW.Permissions == nil {
		jsonW.Permissions = map[string]Permissions{}
	}

	return json.Marshal(jsonW)
}

func (w *HDWallet) UnmarshalJSON(data []byte) error {
	jsonW := &jsonHDWallet{}
	if err := json.Unmarshal(data, jsonW); err != nil {
		return err
	}

	if jsonW.Permissions == nil {
		jsonW.Permissions = map[string]Permissions{}
	}

	*w = HDWallet{
		keyDerivationVersion: jsonW.KeyDerivationVersion,
		node:                 jsonW.Node,
		id:                   jsonW.ID,
		keyRing:              LoadHDKeyRing(jsonW.Keys),
		permissions:          jsonW.Permissions,
	}

	for hostname, perms := range w.permissions {
		if err := ensurePublicKeysPermissionsConsistency(w, perms); err != nil {
			return fmt.Errorf("inconsistent permissions setup for hostname %q: %w", hostname, err)
		}
	}

	if len(w.id) == 0 {
		w.id = walletID(jsonW.Node)
	}

	return nil
}

func (w *HDWallet) deriveKeyNode(nextIndex uint32) (*slip10.Node, error) {
	var derivationFn func(uint32) (*slip10.Node, error)
	switch w.keyDerivationVersion {
	case Version1:
		derivationFn = w.deriveKeyNodeV1
	case Version2:
		derivationFn = w.deriveKeyNodeV2
	default:
		return nil, NewUnsupportedWalletVersionError(w.keyDerivationVersion)
	}

	return derivationFn(nextIndex)
}

func (w *HDWallet) deriveKeyNodeV1(nextIndex uint32) (*slip10.Node, error) {
	keyNode, err := w.node.Derive(OriginIndex + nextIndex)
	if err != nil {
		return nil, fmt.Errorf("couldn't derive key node for index %d: %w", OriginIndex+nextIndex, err)
	}
	return keyNode, nil
}

func (w *HDWallet) deriveKeyNodeV2(nextIndex uint32) (*slip10.Node, error) {
	defaultSubNode, err := w.node.Derive(slip10.FirstHardenedIndex)
	if err != nil {
		return nil, fmt.Errorf("couldn't derive default sub-node: %w", err)
	}
	keyNode, err := defaultSubNode.Derive(slip10.FirstHardenedIndex + nextIndex)
	if err != nil {
		return nil, fmt.Errorf("couldn't derive key node for index %d: %w", OriginIndex+nextIndex, err)
	}
	return keyNode, nil
}

func (w *HDWallet) removeTaintedKeyFromRestrictedKeys(taintedKeyPair HDKeyPair) {
	allKeysAreTainted := w.areAllKeysTainted()

	for hostname, permissions := range w.permissions {
		if !permissions.PublicKeys.Enabled() {
			continue
		}

		if !permissions.PublicKeys.HasRestrictedKeys() {
			// If all the keys in the wallet are tainted, we revoke the
			// permission.
			if allKeysAreTainted {
				permissions.PublicKeys = NoPublicKeysPermission()
				w.permissions[hostname] = permissions
			}
			continue
		}

		restrictedKeys := permissions.PublicKeys.RestrictedKeys

		// Look for the tainted key.
		taintedKeyIdx := -1
		for i, restrictedKey := range restrictedKeys {
			if restrictedKey == taintedKeyPair.PublicKey() {
				taintedKeyIdx = i
				break
			}
		}

		// No tainted key was found, next.
		if taintedKeyIdx == -1 {
			continue
		}

		lastItemIdx := len(restrictedKeys) - 1

		// If lastItemIdx is 0, it means we have a single restricted key, and it
		// is tainted. Removing it will make the slice empty.
		// The user had a clear intent to restrict the access to this single
		// key, we should void the access to public keys and let the third-party
		// application request new permissions.
		// This seems to be the least surprising behavior.
		if lastItemIdx == 0 {
			permissions.PublicKeys = NoPublicKeysPermission()
		} else {
			// We remove the key from the slice.
			if taintedKeyIdx < lastItemIdx {
				copy(restrictedKeys[taintedKeyIdx:], restrictedKeys[taintedKeyIdx+1:])
			}
			restrictedKeys[lastItemIdx] = ""
			restrictedKeys = restrictedKeys[:lastItemIdx]
			permissions.PublicKeys.RestrictedKeys = restrictedKeys
		}

		w.permissions[hostname] = permissions
	}
}

func (w *HDWallet) areAllKeysTainted() bool {
	for _, keyPair := range w.keyRing.ListKeyPairs() {
		if !keyPair.IsTainted() {
			return false
		}
	}
	return true
}

type jsonHDWallet struct {
	// The wallet name is retrieved from the file name it is stored in, so no
	// need to serialize it.

	KeyDerivationVersion uint32                 `json:"version"`
	Node                 *slip10.Node           `json:"node,omitempty"`
	ID                   string                 `json:"id,omitempty"`
	Keys                 []HDKeyPair            `json:"keys"`
	Permissions          map[string]Permissions `json:"permissions"`
}

// NewRecoveryPhrase generates a recovery phrase with an entropy of 256 bits.
func NewRecoveryPhrase() (string, error) {
	entropy, err := bip39.NewEntropy(MaxEntropyByteSize)
	if err != nil {
		return "", fmt.Errorf("couldn't create new wallet: %w", err)
	}
	recoveryPhrase, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("couldn't create recovery phrase: %w", err)
	}

	sanitizedRecoveryPhrase := sanitizeRecoveryPhrase(recoveryPhrase)

	if recoveryPhrase != sanitizedRecoveryPhrase {
		panic("The format of the recovery phrase changed in the bip39 we are using. This may cause problems for the key generation if we import a recovery phrase that is not what the bip39 library is expecting.")
	}

	return recoveryPhrase, nil
}

// sanitizeRecoveryPhrase ensures the recovery phrase always has the right
// format:
//
//	(WORD + " ") * 23 + WORD
//
// This format is what our bip39 library is expecting at the moment we write this
// code. If it ever comes to change, this need to be carefully tested.
func sanitizeRecoveryPhrase(originalRecoveryPhrase string) string {
	return strings.Join(strings.Fields(originalRecoveryPhrase), " ")
}

func deriveWalletNodeFromRecoveryPhrase(recoveryPhrase string) (*slip10.Node, error) {
	seed := bip39.NewSeed(recoveryPhrase, "")
	masterNode, err := slip10.NewMasterNode(seed)
	if err != nil {
		return nil, fmt.Errorf("couldn't create master node: %w", err)
	}
	walletNode, err := masterNode.Derive(OriginIndex)
	if err != nil {
		return nil, fmt.Errorf("couldn't derive wallet node: %w", err)
	}
	return walletNode, nil
}

func walletID(walletNode *slip10.Node) string {
	pubKey, _ := walletNode.Keypair()
	return hex.EncodeToString(pubKey)
}

func ensurePublicKeysPermissionsConsistency(w *HDWallet, perms Permissions) error {
	if perms.PublicKeys.Access == NoAccess {
		if perms.PublicKeys.HasRestrictedKeys() {
			return ErrCannotSetRestrictedKeysWithNoAccess
		}
		return nil
	}

	existingKeys := w.ListKeyPairs()
	if len(existingKeys) == 0 {
		return ErrWalletDoesNotHaveKeys
	}

	if !perms.PublicKeys.HasRestrictedKeys() && w.areAllKeysTainted() {
		return ErrAllKeysInWalletAreTainted
	}

	for _, restrictedKey := range perms.PublicKeys.RestrictedKeys {
		if err := ensureRestrictedKeyIsValid(restrictedKey, existingKeys); err != nil {
			return err
		}
	}

	return nil
}

func ensureRestrictedKeyIsValid(restrictedKey string, existingKeys []KeyPair) error {
	for _, k := range existingKeys {
		if k.PublicKey() == restrictedKey {
			if k.IsTainted() {
				return fmt.Errorf("this restricted key %s is tainted", restrictedKey)
			}
			return nil
		}
	}
	return fmt.Errorf("this restricted key %s does not exist on wallet", restrictedKey)
}
