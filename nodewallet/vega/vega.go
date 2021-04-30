package vega

import (
	"encoding/hex"
	"fmt"
	"path/filepath"

	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/crypto"
)

const (
	defaultVegaWalletOwner = "vega-node"
)

type Wallet struct {
	kp     *wallet.Keypair
	pubKey []byte
}

func DevInit(path, passphrase string) (string, error) {
	fullpath := filepath.Join(path, defaultVegaWalletOwner)
	if ok, _ := fsutil.PathExists(fullpath); ok {
		return "", fmt.Errorf("dev vega wallet already exists at path %v", path)
	}

	w, err := wallet.CreateWalletFile(fullpath, defaultVegaWalletOwner, passphrase)
	if err != nil {
		return "", fmt.Errorf("failed to create Vega wallet file %s: %w", fullpath, err)
	}

	// gen the keypair
	algo := crypto.NewEd25519()
	kp, err := wallet.GenKeypair(algo.Name())
	if err != nil {
		return "", fmt.Errorf("unable to generate new key pair: %v", err)
	}

	w.Keypairs = append(w.Keypairs, *kp)
	_, err = wallet.WriteWalletFile(w, fullpath, passphrase)
	if err != nil {
		return "", fmt.Errorf("failed to write Vega wallet file: %w", err)
	}

	return fullpath, nil
}

func New(path, passphrase string) (*Wallet, error) {
	wal, err := wallet.ReadWalletFile(path, passphrase)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt wallet: %v", err)
	}

	if len(wal.Keypairs) != 1 {
		return nil, fmt.Errorf("vega wallet for node requires to have max 1 keypairs, found %v", len(wal.Keypairs))
	}

	pubBytes, err := hex.DecodeString(wal.Keypairs[0].Pub)
	if err != nil {
		return nil, fmt.Errorf("failed to decode string: %w", err)
	}

	return &Wallet{
		kp:     &wal.Keypairs[0],
		pubKey: pubBytes,
	}, nil
}

func (w *Wallet) Chain() string {
	return "vega"
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	alg, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate signature algorithm: %v", err)
	}
	return wallet.Sign(alg, w.kp, data)
}

func (w *Wallet) Algo() string {
	return w.kp.Algorithm.Name()
}

func (w *Wallet) Version() uint64 {
	return w.kp.Algorithm.Version()
}

func (w *Wallet) PubKeyOrAddress() []byte {
	return w.pubKey
}
