package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/wallet"
	"code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/spf13/cobra"
)

type walletCommand struct {
	command

	rootPath    string
	walletOwner string
	passphrase  string
	data        string
	sig         string
	pubkey      string
	force       bool
	genRsaKey   bool
	metas       string
	Log         *logging.Logger
}

func (w *walletCommand) Init(c *Cli) {
	w.cli = c
	w.cmd = &cobra.Command{
		Use:   "wallet",
		Short: "The wallet subcommand",
		Long:  "Create and manage wallets",
	}

	genkey := &cobra.Command{
		Use:   "genkey",
		Short: "Generate a new keypair for a wallet",
		Long:  "Generate a new keypair for a wallet, this will implicitly generate a new wallet if none exist for the given name",
		RunE:  w.GenKey,
	}
	genkey.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	genkey.Flags().StringVarP(&w.walletOwner, "name", "n", "", "Name of the wallet to use")
	genkey.Flags().StringVarP(&w.passphrase, "passphrase", "p", "", "Passphrase to access the wallet")
	w.cmd.AddCommand(genkey)

	list := &cobra.Command{
		Use:   "list",
		Short: "List keypairs of a wallet",
		Long:  "List all the keypairs for a given wallet",
		RunE:  w.List,
	}
	list.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	list.Flags().StringVarP(&w.walletOwner, "name", "n", "", "Name of the wallet to use")
	list.Flags().StringVarP(&w.passphrase, "passphrase", "p", "", "Passphrase to access the wallet")
	w.cmd.AddCommand(list)

	sign := &cobra.Command{
		Use:   "sign",
		Short: "Sign a blob of data",
		Long:  "Sign a blob of dara base64 encoded",
		RunE:  w.Sign,
	}
	sign.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	sign.Flags().StringVarP(&w.walletOwner, "name", "n", "", "Name of the wallet to use")
	sign.Flags().StringVarP(&w.passphrase, "passphrase", "p", "", "Passphrase to access the wallet")
	sign.Flags().StringVarP(&w.data, "message", "m", "", "Message to be signed (base64)")
	sign.Flags().StringVarP(&w.pubkey, "pubkey", "k", "", "Public key to be used (hex)")
	w.cmd.AddCommand(sign)

	verify := &cobra.Command{
		Use:   "verify",
		Short: "Verify the signature",
		Long:  "Verify the signature for a blob of data",
		RunE:  w.Verify,
	}
	verify.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	verify.Flags().StringVarP(&w.walletOwner, "name", "n", "", "Name of the wallet to use")
	verify.Flags().StringVarP(&w.passphrase, "passphrase", "p", "", "Passphrase to access the wallet")
	verify.Flags().StringVarP(&w.data, "message", "m", "", "Message to be verified (base64)")
	verify.Flags().StringVarP(&w.sig, "signature", "s", "", "Signature to be verified (base64)")
	verify.Flags().StringVarP(&w.pubkey, "pubkey", "k", "", "Public key to be used (hex)")
	w.cmd.AddCommand(verify)

	taint := &cobra.Command{
		Use:   "taint",
		Short: "Taint a public key",
		Long:  "Taint a public key",
		RunE:  w.Taint,
	}
	taint.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	taint.Flags().StringVarP(&w.walletOwner, "name", "n", "", "Name of the wallet to use")
	taint.Flags().StringVarP(&w.passphrase, "passphrase", "p", "", "Passphrase to access the wallet")
	taint.Flags().StringVarP(&w.pubkey, "pubkey", "k", "", "Public key to be used (hex)")
	w.cmd.AddCommand(taint)

	metas := &cobra.Command{
		Use:   "meta",
		Short: "Add metadata to a public key",
		Long:  "Add a list of metadata to a public key",
		RunE:  w.Metas,
	}
	metas.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	metas.Flags().StringVarP(&w.walletOwner, "name", "n", "", "Name of the wallet to use")
	metas.Flags().StringVarP(&w.passphrase, "passphrase", "p", "", "Passphrase to access the wallet")
	metas.Flags().StringVarP(&w.pubkey, "pubkey", "k", "", "Public key to be used (hex)")
	metas.Flags().StringVarP(&w.metas, "metas", "m", "", `A list of metadata e.g: "primary:true;asset;BTC"`)
	w.cmd.AddCommand(metas)

	service := &cobra.Command{
		Use:   "service",
		Short: "The wallet service",
		Long:  "Run or initialize the wallet service",
	}
	w.cmd.AddCommand(service)

	serviceInit := &cobra.Command{
		Use:   "init",
		Short: "Generate the configuration",
		Long:  "Generate the configuration for the wallet service",
		RunE:  w.ServiceInit,
	}
	serviceInit.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	serviceInit.Flags().BoolVarP(&w.force, "force", "f", false, "Erase exiting wallet service configuration at the specified path")
	serviceInit.Flags().BoolVarP(&w.genRsaKey, "genrsakey", "g", false, "Generate rsa keys for the jwt tokens")
	service.AddCommand(serviceInit)

	serviceRun := &cobra.Command{
		Use:   "run",
		Short: "Start the vega wallet service",
		Long:  "Start a vega wallet service behind an http server",
		RunE:  w.ServiceRun,
	}
	serviceRun.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	service.AddCommand(serviceRun)
}

func (w *walletCommand) ServiceInit(cmd *cobra.Command, args []string) error {
	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	return wallet.GenConfig(w.Log, w.rootPath, w.force, w.genRsaKey)
}

func (w *walletCommand) ServiceRun(cmd *cobra.Command, args []string) error {
	cfg, err := wallet.LoadConfig(w.rootPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := wallet.NewService(w.Log, cfg, w.rootPath)
	if err != nil {
		return err
	}
	go func() {
		defer cancel()
		err := srv.Start()
		if err != nil {
			w.Log.Error("error starting wallet http server", logging.Error(err))
		}
	}()

	waitSig(ctx, w.Log)

	err = srv.Stop()
	if err != nil {
		w.Log.Error("error stopping wallet http server", logging.Error(err))
	} else {
		w.Log.Info("wallet http server stopped with success")
	}

	return nil
}

func (w *walletCommand) GenKey(cmd *cobra.Command, args []string) error {
	if len(w.walletOwner) <= 0 {
		return errors.New("wallet name is required")
	}
	if len(w.passphrase) <= 0 {
		return errors.New("passphrase is required")
	}

	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	if err := wallet.EnsureBaseFolder(w.rootPath); err != nil {
		return fmt.Errorf("unable to initialization root folder: %v", err)
	}

	_, err := wallet.Read(w.rootPath, w.walletOwner, w.passphrase)
	if err != nil {
		if err != wallet.ErrWalletDoesNotExists {
			// this an invalid key, returning error
			return fmt.Errorf("unable to decrypt wallet: %v", err)
		}
		// wallet do not exit, let's try to create it
		_, err = wallet.Create(w.rootPath, w.walletOwner, w.passphrase)
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
	_, err = wallet.AddKeypair(kp, w.rootPath, w.walletOwner, w.passphrase)
	if err != nil {
		return fmt.Errorf("unable to add keypair to wallet: %v", err)
	}

	// print the new keys for user info
	fmt.Printf("new generated keys:\n")
	fmt.Printf("public: %v\n", kp.Pub)
	fmt.Printf("private: %v\n", kp.Priv)

	return nil
}

func (w *walletCommand) List(cmd *cobra.Command, args []string) error {
	if len(w.walletOwner) <= 0 {
		return errors.New("wallet name is required")
	}
	if len(w.passphrase) <= 0 {
		return errors.New("passphrase is required")
	}

	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	wal, err := wallet.Read(w.rootPath, w.walletOwner, w.passphrase)
	if err != nil {
		return fmt.Errorf("unable to decrypt wallet: %v", err)
	}

	buf, err := json.MarshalIndent(wal, " ", " ")
	if err != nil {
		return fmt.Errorf("unable to indent message: %v", err)
	}

	// print the new keys for user info
	fmt.Printf("List of all your keypairs:\n")
	fmt.Printf("%v\n", string(buf))

	return nil
}

func (w *walletCommand) Sign(cmd *cobra.Command, args []string) error {
	if len(w.walletOwner) <= 0 {
		return errors.New("wallet name is required")
	}
	if len(w.passphrase) <= 0 {
		return errors.New("passphrase is required")
	}
	if len(w.pubkey) <= 0 {
		return errors.New("pubkey is required")
	}
	if len(w.data) <= 0 {
		return errors.New("data is required")
	}

	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	wal, err := wallet.Read(w.rootPath, w.walletOwner, w.passphrase)
	if err != nil {
		return fmt.Errorf("unable to decrypt wallet: %v", err)
	}

	dataBuf, err := base64.StdEncoding.DecodeString(w.data)
	if err != nil {
		return fmt.Errorf("invalid base64 encoded data: %v", err)
	}

	var kp *wallet.Keypair
	for i, v := range wal.Keypairs {
		if v.Pub == w.pubkey {
			kp = &wal.Keypairs[i]
		}
	}
	if kp == nil {
		return fmt.Errorf("unknown public key: %v", w.pubkey)
	}
	if kp.Tainted {
		return fmt.Errorf("key is tainted: %v", w.pubkey)
	}

	alg, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	if err != nil {
		return fmt.Errorf("unable to instanciate signature algorithm: %v", err)
	}
	sig, err := wallet.Sign(alg, kp, dataBuf)
	if err != nil {
		return fmt.Errorf("unable to sign: %v", err)
	}
	fmt.Printf("%v\n", base64.StdEncoding.EncodeToString(sig))

	return nil
}

func (w *walletCommand) Verify(cmd *cobra.Command, args []string) error {
	if len(w.walletOwner) <= 0 {
		return errors.New("wallet name is required")
	}
	if len(w.passphrase) <= 0 {
		return errors.New("passphrase is required")
	}
	if len(w.pubkey) <= 0 {
		return errors.New("pubkey is required")
	}
	if len(w.data) <= 0 {
		return errors.New("data is required")
	}
	if len(w.sig) <= 0 {
		return errors.New("data is required")
	}

	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	wal, err := wallet.Read(w.rootPath, w.walletOwner, w.passphrase)
	if err != nil {
		return fmt.Errorf("unable to decrypt wallet: %v", err)
	}

	dataBuf, err := base64.StdEncoding.DecodeString(w.data)
	if err != nil {
		return fmt.Errorf("invalid base64 encoded data: %v", err)
	}
	sigBuf, err := base64.StdEncoding.DecodeString(w.sig)
	if err != nil {
		return fmt.Errorf("invalid base64 encoded data: %v", err)
	}

	var kp *wallet.Keypair
	for i, v := range wal.Keypairs {
		if v.Pub == w.pubkey {
			kp = &wal.Keypairs[i]
		}
	}
	if kp == nil {
		return fmt.Errorf("unknown public key: %v", w.pubkey)
	}

	alg, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	if err != nil {
		return fmt.Errorf("unable to instanciate signature algorithm: %v", err)
	}
	verified, err := wallet.Verify(alg, kp, dataBuf, sigBuf)
	if err != nil {
		return fmt.Errorf("unable to verify: %v", err)
	}
	fmt.Printf("%v\n", verified)

	return nil
}

func (w *walletCommand) Taint(cmd *cobra.Command, args []string) error {
	if len(w.walletOwner) <= 0 {
		return errors.New("wallet name is required")
	}
	if len(w.passphrase) <= 0 {
		return errors.New("passphrase is required")
	}

	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	wal, err := wallet.Read(w.rootPath, w.walletOwner, w.passphrase)
	if err != nil {
		return fmt.Errorf("unable to decrypt wallet: %v", err)
	}

	var kp *wallet.Keypair
	for i, v := range wal.Keypairs {
		if v.Pub == w.pubkey {
			kp = &wal.Keypairs[i]
		}
	}
	if kp == nil {
		return fmt.Errorf("unknown public key: %v", w.pubkey)
	}

	if kp.Tainted {
		return fmt.Errorf("key %v is already tainted", w.pubkey)
	}

	kp.Tainted = true

	_, err = wallet.Write(wal, w.rootPath, w.walletOwner, w.passphrase)
	if err != nil {
		return err
	}

	return nil
}

func (w *walletCommand) Metas(cmd *cobra.Command, args []string) error {
	if len(w.walletOwner) <= 0 {
		return errors.New("wallet name is required")
	}
	if len(w.passphrase) <= 0 {
		return errors.New("passphrase is required")
	}
	if len(w.pubkey) <= 0 {
		return errors.New("pubkey is required")
	}
	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	wal, err := wallet.Read(w.rootPath, w.walletOwner, w.passphrase)
	if err != nil {
		return fmt.Errorf("unable to decrypt wallet: %v", err)
	}

	var kp *wallet.Keypair
	for i, v := range wal.Keypairs {
		if v.Pub == w.pubkey {
			kp = &wal.Keypairs[i]
		}
	}
	if kp == nil {
		return fmt.Errorf("unknown public key: %v", w.pubkey)
	}

	var meta []wallet.Meta
	if len(w.metas) > 0 {
		// expect ; separated metas
		metasSplit := strings.Split(w.metas, ";")
		for _, v := range metasSplit {
			metaVal := strings.Split(v, ":")
			if len(metaVal) != 2 {
				return fmt.Errorf("invalid meta format")
			}
			meta = append(meta, wallet.Meta{Key: metaVal[0], Value: metaVal[1]})
		}

	}

	kp.Meta = meta
	_, err = wallet.Write(wal, w.rootPath, w.walletOwner, w.passphrase)
	if err != nil {
		return err
	}

	return nil
}
