package nodewallet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/wallet/crypto"
)

type WalletConfig struct {
	Chain      string `json:"chain"`
	Path       string `json:"wallet_path"`
	Passphrase string `json:"passphrase"`
}

type store struct {
	Wallets []WalletConfig
}

func (s *store) AddWallet(w WalletConfig) {
	for i, v := range s.Wallets {
		if v.Chain == w.Chain {
			s.Wallets[i] = w
			return
		}
	}
	s.Wallets = append(s.Wallets, w)
}

func loadStore(path string, passphrase string) (*store, error) {
	// make sure this do not exists already
	if ok, err := fsutil.PathExists(path); !ok {
		return nil, fmt.Errorf("unable to load store (%v)", err)
	}

	// read file
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read store file (%v)", err)
	}

	// decrypt the buffer
	decBuf, err := crypto.Decrypt(buf, passphrase)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt store file (%v)", err)
	}

	// unmarshal the wallet now an return
	stor := &store{}
	return stor, json.Unmarshal(decBuf, stor)
}

func saveStore(stor *store, path, passphrase string) error {
	// marshal our wallet
	buf, err := json.Marshal(stor)
	if err != nil {
		return err
	}

	encBuf, err := crypto.Encrypt(buf, passphrase)
	if err != nil {
		return fmt.Errorf("unable to encrypt store file (%v)", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	f.Write(encBuf)
	f.Close()

	return nil
}
