package wallet

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ed25519"

	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/wallet/crypto"
)

var (
	ErrWalletAlreadyExist = errors.New("a wallet with the same name already exist")
	ErrWalletDoesNotExist = errors.New("wallet does not exist")
)

const (
	walletBaseFolder = "wallets"
)

type Wallet struct {
	Owner    string
	Keypairs []Keypair
}

type Keypair struct {
	Pub       string
	Priv      string
	Algorithm crypto.SignatureAlgorithm
	Tainted   bool
	Meta      []Meta

	// byte version of the public and private keys
	// not being marshalled/sent over the network
	// or saved into the wallet file.
	pubBytes  []byte
	privBytes []byte
}

func (k *Keypair) MarshalJSON() ([]byte, error) {
	k.Pub = hex.EncodeToString(k.pubBytes)
	k.Priv = hex.EncodeToString(k.privBytes)
	type alias Keypair
	aliasKeypair := (*alias)(k)
	return json.Marshal(aliasKeypair)
}

func (k *Keypair) UnmarshalJSON(data []byte) error {
	type alias Keypair
	aliasKeypair := (*alias)(k)
	if err := json.Unmarshal(data, aliasKeypair); err != nil {
		return err
	}
	var err error
	k.pubBytes, err = hex.DecodeString(k.Pub)
	if err != nil {
		return err
	}
	k.privBytes, err = hex.DecodeString(k.Priv)
	return nil
}

type Meta struct {
	Key   string
	Value string
}

func New(owner string) Wallet {
	return Wallet{
		Owner:    owner,
		Keypairs: []Keypair{},
	}
}

func GenKeypair(algorithm string) (*Keypair, error) {
	algo, err := crypto.NewSignatureAlgorithm(algorithm)
	if err != nil {
		return nil, err
	}
	pub, priv, err := algo.GenKey()
	if err != nil {
		return nil, err
	}

	privBytes := priv.(ed25519.PrivateKey)
	pubBytes := pub.(ed25519.PublicKey)
	return &Keypair{
		Priv:      hex.EncodeToString(privBytes),
		Pub:       hex.EncodeToString(pubBytes),
		Algorithm: algo,
		privBytes: privBytes,
		pubBytes:  pubBytes,
	}, err

}

func NewKeypair(algo crypto.SignatureAlgorithm, pub, priv []byte) Keypair {
	return Keypair{
		Algorithm: algo,
		pubBytes:  pub,
		privBytes: priv,
	}
}

func EnsureBaseFolder(root string) error {
	return fsutil.EnsureDir(filepath.Join(root, walletBaseFolder))
}

func Create(root, owner, passphrase string) (*Wallet, error) {
	w := Wallet{
		Owner: owner,
	}

	// build walletpath
	walletpath := filepath.Join(root, walletBaseFolder, owner)

	// make sure this do not exists already
	if ok, _ := fsutil.PathExists(walletpath); ok {
		return nil, ErrWalletAlreadyExist
	}

	return writeWallet(&w, root, owner, passphrase)
}

func AddKeypair(kp *Keypair, root, owner, passphrase string) (*Wallet, error) {
	w, err := Read(root, owner, passphrase)
	if err != nil {
		return nil, err
	}

	w.Keypairs = append(w.Keypairs, *kp)

	return writeWallet(w, root, owner, passphrase)
}

func Read(root, owner, passphrase string) (*Wallet, error) {
	// build walletpath
	walletpath := filepath.Join(root, walletBaseFolder, owner)

	// make sure this do not exists already
	if ok, _ := fsutil.PathExists(walletpath); !ok {
		return nil, ErrWalletDoesNotExist
	}

	// read file
	buf, err := ioutil.ReadFile(walletpath)
	if err != nil {
		return nil, err
	}

	// decrypt the buffer
	decBuf, err := crypto.Decrypt(buf, passphrase)
	if err != nil {
		return nil, err
	}

	// unmarshal the wallet now an return
	w := &Wallet{}
	return w, json.Unmarshal(decBuf, w)
}

func writeWallet(w *Wallet, root, owner, passphrase string) (*Wallet, error) {
	// build walletpath
	walletpath := filepath.Join(root, walletBaseFolder, owner)

	// marshal our wallet
	buf, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	// encrypt our data
	encBuf, err := crypto.Encrypt(buf, passphrase)
	if err != nil {
		return nil, err
	}

	// create and write file
	f, _ := os.Create(walletpath)
	f.Write(encBuf)
	f.Close()

	return w, nil
}
