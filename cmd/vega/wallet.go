package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/crypto"
	"github.com/jessevdk/go-flags"
)

type WalletCmd struct {
	config.RootPathFlag

	Genkey  walletGenkey  `command:"genkey" description:"Generates a new keypar for a wallet" long-description:"Generate a new keypair for a wallet, this will implicitly generate a new wallet if none exist for the given name"`
	List    walletList    `command:"list" description:"Lists keypairs of a wallet" long-description:"Lists all the keypairs for a given wallet"`
	Sign    walletSign    `command:"sign" description:"Signs (base64 encoded) data" long-description:"Signs (base64 encoded) data given a public key"`
	Verify  walletVerify  `command:"verify" description:"Verifies a signature" long-description:"Verifies a signature for a given data"`
	Taint   walletTaint   `command:"taint" description:"Taints a public key" long-description:"Taints a public key"`
	Meta    walletMeta    `command:"meta" description:"Adds metadata to a public key" long-description:"Adds a list of metadata to a public key"`
	Service walletService `command:"service" description:"The wallet service" long-description:"Runs or initializes the wallet service"`
}

// walletCmd is a global variable that holds generic options for the wallet
// sub-commands.
var walletCmd WalletCmd

func Wallet(ctx context.Context, parser *flags.Parser) error {
	// Build the walletCmd with default values and ctx where needed.
	walletCmd = WalletCmd{
		RootPathFlag: config.NewRootPathFlag(),
		Service: walletService{
			Run: walletServiceRun{
				ctx:    ctx,
				Config: wallet.NewDefaultConfig(),
			},
		},
	}

	_, err := parser.AddCommand("wallet", "Create and manage wallets", "", &walletCmd)
	return err
}

func readWallet(rootPath, name, pass string) (*wallet.Wallet, error) {
	if ok, err := fsutil.PathExists(rootPath); !ok {
		return nil, fmt.Errorf("invalid root directory path: %w", err)
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
	config.PassphraseFlag
	Name string `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
}

func (opts *walletGenkey) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.Passphrase.Get(name)
	if err != nil {
		return err
	}

	if _, err := readWallet(walletCmd.RootPath, name, pass); err != nil {
		if !errors.Is(err, wallet.ErrWalletDoesNotExists) {
			// this an invalid key, returning error
			return err
		}
		// wallet do not exit, let's try to create it
		_, err = wallet.Create(walletCmd.RootPath, name, pass)
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
	_, err = wallet.AddKeypair(kp, walletCmd.RootPath, opts.Name, pass)
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
	config.PassphraseFlag
	Name string `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
}

func (opts *walletList) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.Passphrase.Get(name)
	if err != nil {
		return err
	}

	w, err := readWallet(walletCmd.RootPath, name, pass)
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
	config.PassphraseFlag
	Name    string          `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
	Message encoding.Base64 `short:"m" long:"message" description:"Message to be signed (base64 encoded)" required:"true"`
	PubKey  string          `short:"k" long:"pubkey" description:"Public key to be used (hex encoded)" required:"true"`
}

func (opts *walletSign) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.Passphrase.Get(name)
	if err != nil {
		return err
	}

	w, err := readWallet(walletCmd.RootPath, name, pass)
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
		return fmt.Errorf("unable to instantiate signature algorithm: %v", err)
	}
	sig, err := wallet.Sign(alg, kp, opts.Message)
	if err != nil {
		return fmt.Errorf("unable to sign: %v", err)
	}
	fmt.Printf("%v\n", base64.StdEncoding.EncodeToString(sig))

	return nil
}

type walletVerify struct {
	config.PassphraseFlag
	Name    string          `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
	Message encoding.Base64 `short:"m" long:"message" description:"Message to be signed (base64 encoded)" required:"true"`
	PubKey  string          `short:"k" long:"pubkey" description:"Public key to be used (hex encoded)" required:"true"`
	Sig     encoding.Base64 `short:"s" long:"signature" description:"Signature to be verified (base64 encoded)" required:"true"`
}

func (opts *walletVerify) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.Passphrase.Get(name)
	if err != nil {
		return err
	}

	w, err := readWallet(walletCmd.RootPath, name, pass)
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
		return fmt.Errorf("unable to instantiate signature algorithm: %v", err)
	}
	verified, err := wallet.Verify(alg, kp, opts.Message, opts.Sig)
	if err != nil {
		return fmt.Errorf("unable to verify: %v", err)
	}
	fmt.Printf("%v\n", verified)

	return nil
}

type walletTaint struct {
	config.PassphraseFlag
	Name   string `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
	PubKey string `short:"k" long:"pubkey" description:"Public key to be used (hex encoded)" required:"true"`
}

func (opts *walletTaint) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.Passphrase.Get(name)
	if err != nil {
		return err
	}

	w, err := readWallet(walletCmd.RootPath, name, pass)
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

	_, err = wallet.Write(w, walletCmd.RootPath, name, pass)
	return err
}

type walletMeta struct {
	config.PassphraseFlag
	Name   string `short:"n" long:"name" description:"Name of the wallet to user" required:"true"`
	PubKey string `short:"k" long:"pubkey" description:"Public key to be used (hex encoded)" required:"true"`
	Metas  string `short:"m" long:"metas" description:"A list of metadata e.g:'primary:true;asset:BTC'" required:"true"`
}

func (opts *walletMeta) Execute(_ []string) error {
	name := opts.Name
	pass, err := opts.Passphrase.Get(name)
	if err != nil {
		return err
	}

	w, err := readWallet(walletCmd.RootPath, name, pass)
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

	var meta []wallet.Meta
	if len(opts.Metas) > 0 {
		// expect ; separated metas
		split := strings.Split(opts.Metas, ";")
		for _, v := range split {
			val := strings.Split(v, ":")
			if len(val) != 2 {
				return fmt.Errorf("invalid meta format")
			}
			meta = append(meta, wallet.Meta{Key: val[0], Value: val[1]})
		}

	}
	kp.Meta = meta

	_, err = wallet.Write(w, walletCmd.RootPath, name, pass)
	return err
}

type walletServiceInit struct {
	Force  bool `short:"f" long:"force" description:"Erase existing configuration at specified path"`
	GenRSA bool `short:"g" long:"genrsakey" description:"Generates RSA for the JWT tokens"`
}

func (opts *walletServiceInit) Execute(_ []string) error {
	if ok, err := fsutil.PathExists(walletCmd.RootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	return wallet.GenConfig(log, walletCmd.RootPath, opts.Force, opts.GenRSA)
}

type walletServiceRun struct {
	ctx    context.Context
	Config wallet.Config
}

func (opts *walletServiceRun) Execute(_ []string) error {
	cfg, err := wallet.LoadConfig(walletCmd.RootPath)
	if err != nil {
		return err
	}

	opts.Config = *cfg
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	srv, err := wallet.NewService(log, &opts.Config, walletCmd.RootPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(opts.ctx)
	go func() {
		defer cancel()
		err := srv.Start()
		if err != nil {
			log.Error("error starting wallet http server", logging.Error(err))
		}
	}()

	waitSig(ctx, log)

	err = srv.Stop()
	if err != nil {
		log.Error("error stopping wallet http server", logging.Error(err))
	} else {
		log.Info("wallet http server stopped with success")
	}

	return nil
}

type walletService struct {
	Init walletServiceInit `command:"init" description:"Generates the configuration" long-description:"Generates the configuration for the wallet service"`
	Run  walletServiceRun  `command:"run" description:"Start the vega wallet service" long-description:"Start a vega wallet service behind an http server"`
}
