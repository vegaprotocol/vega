package bridge

import (
	"encoding/hex"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/bridges"
	"code.vegaprotocol.io/vega/config"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/types/num"
)

type ERC20Cmd struct {
	config.RootPathFlag
	config.PassphraseFlag

	AddSigner    ERC20AddSignerCmd    `command:"add_signer" description:"Create signature to add a new signer to the erc20 bridge"`
	RemoveSigner ERC20RemoveSignerCmd `command:"remove_signer" description:"Create signature to remove a signer from the erc20 bridge"`
	SetThreshold ERC20SetThresholdCmd `command:"set_threshold" description:"Create signature to change the threshold of required signature to apply changes to the bridge"`
}

var erc20Cmd *ERC20Cmd

func ERC20() *ERC20Cmd {
	root := config.NewRootPathFlag()
	erc20Cmd = &ERC20Cmd{
		RootPathFlag: root,
		AddSigner: ERC20AddSignerCmd{
			Config: nodewallet.NewDefaultConfig(),
		},
		RemoveSigner: ERC20RemoveSignerCmd{
			Config: nodewallet.NewDefaultConfig(),
		},
		SetThreshold: ERC20SetThresholdCmd{
			Config: nodewallet.NewDefaultConfig(),
		},
	}

	return erc20Cmd
}

type ERC20AddSignerCmd struct {
	Config nodewallet.Config

	NewSigner string `long:"new-signer" required:"true" description:"Ethereum address of the new signer"`
	Submitter string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce     string `long:"nonce" required:"true" description:"An nonce for this signature"`
}

func (opts *ERC20AddSignerCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	if ok, err := vgfs.PathExists(erc20Cmd.RootPath); !ok {
		return fmt.Errorf("invalid root directory path: %w", err)
	}

	pass, err := erc20Cmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	conf, err := config.Read(erc20Cmd.RootPath)
	if err != nil {
		return err
	}
	opts.Config = conf.NodeWallet

	nw, err := nodewallet.New(log, conf.NodeWallet, pass, nil, erc20Cmd.RootPath)
	if err != nil {
		return err
	}

	w, ok := nw.Get(nodewallet.Blockchain("ethereum"))
	if !ok {
		return errors.New("no ethereum wallet configured")
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	multiSigControl := bridges.NewERC20MultiSigControl(w)
	_, signature, err := multiSigControl.AddSigner(
		opts.NewSigner, opts.Submitter, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", hex.EncodeToString(signature))
	return nil
}

type ERC20RemoveSignerCmd struct {
	Config    nodewallet.Config
	OldSigner string `long:"old-signer" required:"true" description:"Ethereum address of signer to remove"`
	Submitter string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce     string `long:"nonce" required:"true" description:"An nonce for this signature"`
}

func (opts *ERC20RemoveSignerCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	if ok, err := vgfs.PathExists(erc20Cmd.RootPath); !ok {
		return fmt.Errorf("invalid root directory path: %w", err)
	}

	pass, err := erc20Cmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	conf, err := config.Read(erc20Cmd.RootPath)
	if err != nil {
		return err
	}
	opts.Config = conf.NodeWallet

	nw, err := nodewallet.New(log, conf.NodeWallet, pass, nil, erc20Cmd.RootPath)
	if err != nil {
		return err
	}

	w, ok := nw.Get(nodewallet.Blockchain("ethereum"))
	if !ok {
		return errors.New("no ethereum wallet configured")
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	multiSigControl := bridges.NewERC20MultiSigControl(w)
	_, signature, err := multiSigControl.RemoveSigner(
		opts.OldSigner, opts.Submitter, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", hex.EncodeToString(signature))
	return nil
}

type ERC20SetThresholdCmd struct {
	Config       nodewallet.Config
	NewThreshold uint16 `long:"new-threshold" required:"true" description:"The new threshold to be used on the bridge"`
	Submitter    string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce        string `long:"nonce" required:"true" description:"An nonce for this signature"`
}

func (opts *ERC20SetThresholdCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	if opts.NewThreshold == 0 || opts.NewThreshold > 1000 {
		return fmt.Errorf("invalid new threshold, required to be > 0 and <= 1000, got %d", opts.NewThreshold)
	}

	if ok, err := vgfs.PathExists(erc20Cmd.RootPath); !ok {
		return fmt.Errorf("invalid root directory path: %w", err)
	}

	pass, err := erc20Cmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	conf, err := config.Read(erc20Cmd.RootPath)
	if err != nil {
		return err
	}
	opts.Config = conf.NodeWallet

	nw, err := nodewallet.New(log, conf.NodeWallet, pass, nil, erc20Cmd.RootPath)
	if err != nil {
		return err
	}

	w, ok := nw.Get(nodewallet.Blockchain("ethereum"))
	if !ok {
		return errors.New("no ethereum wallet configured")
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	multiSigControl := bridges.NewERC20MultiSigControl(w)
	_, signature, err := multiSigControl.SetThreshold(
		opts.NewThreshold, opts.Submitter, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", hex.EncodeToString(signature))
	return nil
}
