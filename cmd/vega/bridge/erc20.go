package bridge

import (
	"encoding/hex"
	"errors"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/bridges"
	"code.vegaprotocol.io/vega/config"
	nodewallet "code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/types/num"
)

type ERC20Cmd struct {
	config.VegaHomeFlag
	config.PassphraseFlag

	AddSigner    ERC20AddSignerCmd    `command:"add_signer" description:"Create signature to add a new signer to the erc20 bridge"`
	RemoveSigner ERC20RemoveSignerCmd `command:"remove_signer" description:"Create signature to remove a signer from the erc20 bridge"`
	SetThreshold ERC20SetThresholdCmd `command:"set_threshold" description:"Create signature to change the threshold of required signature to apply changes to the bridge"`
}

var erc20Cmd *ERC20Cmd

func ERC20() *ERC20Cmd {
	erc20Cmd = &ERC20Cmd{
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
	Nonce     string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20AddSignerCmd) Execute(_ []string) error {
	pass, err := erc20Cmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	vegaPaths := paths.NewPaths(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	w, err := nodewallet.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
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
	Nonce     string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20RemoveSignerCmd) Execute(_ []string) error {
	pass, err := erc20Cmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	vegaPaths := paths.NewPaths(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	w, err := nodewallet.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
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
	Nonce        string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20SetThresholdCmd) Execute(_ []string) error {
	if opts.NewThreshold == 0 || opts.NewThreshold > 1000 {
		return fmt.Errorf("invalid new threshold, required to be > 0 and <= 1000, got %d", opts.NewThreshold)
	}

	pass, err := erc20Cmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	vegaPaths := paths.NewPaths(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	w, err := nodewallet.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
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
