package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/crypto"
	"github.com/jessevdk/go-flags"
)

func readWallet(rootPath, name, pass string) (*wallet.Wallet, error) {
	if ok, err := fsutil.PathExists(rootPath); !ok {
		return nil, fmt.Errorf("invalid root directory path: %v", err)
	}

	if err := wallet.EnsureBaseFolder(rootPath); err != nil {
		return nil, err
	}

	w, err := wallet.Read(rootPath, name, pass)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt wallet: %w", err)
	}
	return w, nil
}

type walletGenkey struct {
	RootPathOption
	PassphraseOption
	Name string `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
}

func (opts *walletGenkey) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.PassphraseOption.Get(name)
	if err != nil {
		return err
	}

	if _, err := readWallet(opts.RootPath, name, pass); err != nil {
		if !errors.Is(err, wallet.ErrWalletDoesNotExists) {
			// this an invalid key, returning error
			return err
		}
		// wallet do not exit, let's try to create it
		_, err = wallet.Create(opts.RootPath, name, pass)
		if err != nil {
			return fmt.Errorf("unable to create wallet: %v", err)
		}
	}

	// at this point we have a valid wallet
	// let's generate the keypair
	// defaulting to ed25519 for now
	algo := crypto.NewEd25519()
	kp, err := wallet.GenKeypair(algo.Name())
	if err != nil {
		return fmt.Errorf("unable to generate new key pair: %v", err)
	}

	// now updating the wallet and saving it
	_, err = wallet.AddKeypair(kp, opts.RootPath, opts.Name, pass)
	if err != nil {
		return fmt.Errorf("unable to add keypair to wallet: %v", err)
	}

	// print the new keys for user info
	fmt.Printf("new generated keys:\n")
	fmt.Printf("public: %v\n", kp.Pub)
	fmt.Printf("private: %v\n", kp.Priv)

	return nil
}

type walletList struct {
	RootPathOption
	PassphraseOption
	Name string `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
}

func (opts *walletList) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.PassphraseOption.Get(name)
	if err != nil {
		return err
	}

	w, err := readWallet(opts.RootPath, name, pass)
	if err != nil {
		return err
	}

	buf, err := json.MarshalIndent(w, " ", " ")
	if err != nil {
		return fmt.Errorf("unable to indent message: %v", err)
	}

	// print the new keys for user info
	fmt.Printf("%v\n", string(buf))
	return nil
}

type walletSign struct {
	RootPathOption
	PassphraseOption
	Name    string          `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
	Message encoding.Base64 `short:"m" long:"message" description:"Message to be signed (base64 encoded)" required:"true"`
	PubKey  string          `short:"k" long:"pubkey" description:"Public key to be used (hex encoded)" required:"true"`
}

func (opts *walletSign) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.PassphraseOption.Get(name)
	if err != nil {
		return err
	}

	w, err := readWallet(opts.RootPath, name, pass)
	if err != nil {
		return err
	}

	var kp *wallet.Keypair
	for i, v := range w.Keypairs {
		if v.Pub == opts.PubKey {
			kp = &w.Keypairs[i]
			break
		}
	}
	if kp == nil {
		return fmt.Errorf("unknown public key: %v", opts.PubKey)
	}
	if kp.Tainted {
		return fmt.Errorf("key is tainted: %v", opts.PubKey)
	}

	alg, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	if err != nil {
		return fmt.Errorf("unable to instanciate signature algorithm: %v", err)
	}
	sig, err := wallet.Sign(alg, kp, opts.Message)
	if err != nil {
		return fmt.Errorf("unable to sign: %v", err)
	}
	fmt.Printf("%v\n", base64.StdEncoding.EncodeToString(sig))

	return nil
}

type walletVerify struct {
	RootPathOption
	PassphraseOption
	Name    string          `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
	Message encoding.Base64 `short:"m" long:"message" description:"Message to be signed (base64 encoded)" required:"true"`
	PubKey  string          `short:"k" long:"pubkey" description:"Public key to be used (hex encoded)" required:"true"`
	Sig     encoding.Base64 `short:"s" long:"signature" description:"Signature to be verified (base64 encoded)" required:"true"`
}

func (opts *walletVerify) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.PassphraseOption.Get(name)
	if err != nil {
		return err
	}

	w, err := readWallet(opts.RootPath, name, pass)
	if err != nil {
		return err
	}

	var kp *wallet.Keypair
	for i, v := range w.Keypairs {
		if v.Pub == opts.PubKey {
			kp = &w.Keypairs[i]
			break
		}
	}
	if kp == nil {
		return fmt.Errorf("unknown public key: %v", opts.PubKey)
	}

	alg, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	if err != nil {
		return fmt.Errorf("unable to instanciate signature algorithm: %v", err)
	}
	verified, err := wallet.Verify(alg, kp, opts.Message, opts.Sig)
	if err != nil {
		return fmt.Errorf("unable to verify: %v", err)
	}
	fmt.Printf("%v\n", verified)

	return nil
}

type walletTaint struct {
	RootPathOption
	PassphraseOption
	Name   string `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
	PubKey string `short:"k" long:"pubkey" description:"Public key to be used (hex encoded)" required:"true"`
}

func (opts *walletTaint) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.PassphraseOption.Get(name)
	if err != nil {
		return err
	}

	w, err := readWallet(opts.RootPath, name, pass)
	if err != nil {
		return err
	}
	var kp *wallet.Keypair
	for i, v := range w.Keypairs {
		if v.Pub == opts.PubKey {
			kp = &w.Keypairs[i]
			break
		}
	}
	if kp == nil {
		return fmt.Errorf("unknown public key: %v", opts.PubKey)
	}

	if kp.Tainted {
		return fmt.Errorf("key %v is already tainted", opts.PubKey)
	}

	kp.Tainted = true

	_, err = wallet.Write(w, opts.RootPath, name, pass)
	return err
}

func Wallet(parser *flags.Parser) error {
	root, err := parser.AddCommand("wallet", "Create and manage wallets", "", &Empty{})
	if err != nil {
		return err
	}

	cmds := []struct {
		name  string
		short string
		long  string
		opts  interface{}
	}{
		{
			"genkey",
			"Generates a new keypar for a wallet",
			"Generate a new keypair for a wallet, this will implicitly generate a new wallet if none exist for the given name",
			&walletGenkey{
				RootPathOption: NewRootPathOption(),
			},
		},
		{
			"list",
			"Lists keypairs of a wallet",
			"Lists all the keypairs for a given wallet",
			&walletList{
				RootPathOption: NewRootPathOption(),
			},
		},
		{
			"sign",
			"Signs (base64 encoded) data",
			"Signs (base64 encoded) data given a public key",
			&walletSign{
				RootPathOption: NewRootPathOption(),
			},
		},
		{
			"verify",
			"Verifies a signature",
			"Verifies a signature for a given data",
			&walletVerify{
				RootPathOption: NewRootPathOption(),
			},
		},
	}

	for _, cmd := range cmds {
		if _, err := root.AddCommand(cmd.name, cmd.short, cmd.long, cmd.opts); err != nil {
			return err
		}
	}
	return nil
}
