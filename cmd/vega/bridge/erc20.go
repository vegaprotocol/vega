package bridge

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/bridges"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/jessevdk/go-flags"
)

type ERC20Cmd struct {
	config.VegaHomeFlag
	config.PassphraseFlag

	AddSigner         ERC20AddSignerCmd         `command:"add_signer" description:"Create signature to add a new signer to the erc20 bridge"`
	RemoveSigner      ERC20RemoveSignerCmd      `command:"remove_signer" description:"Create signature to remove a signer from the erc20 bridge"`
	SetThreshold      ERC20SetThresholdCmd      `command:"set_threshold" description:"Create signature to change the threshold of required signature to apply changes to the bridge"`
	ListAsset         ERC20ListAssetCmd         `command:"list_asset" description:"Add a new erc20 asset to the erc20 bridge"`
	RemoveAsset       ERC20RemoveAssetCmd       `command:"remove_asset" description:"Remove an erc20 asset from the erc20 bridge"`
	WithdrawAsset     ERC20WithdrawAssetCmd     `command:"withdraw_asset" description:"Withdraw ERC20 from the bridge"`
	SetDepositMinimum ERC20SetDepositMinimumCmd `command:"set_deposit_minimum" description:"Set the minimum allowed deposit for an ERC20 token on the bridge"`
	SetDepositMaximum ERC20SetDepositMaximumCmd `command:"set_deposit_maximum" description:"Set the maximum allowed deposit for an ERC20 token on the bridge"`
}

var erc20Cmd *ERC20Cmd

func ERC20() *ERC20Cmd {
	erc20Cmd = &ERC20Cmd{
		AddSigner: ERC20AddSignerCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		RemoveSigner: ERC20RemoveSignerCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		SetThreshold: ERC20SetThresholdCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		ListAsset: ERC20ListAssetCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		RemoveAsset: ERC20RemoveAssetCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		WithdrawAsset: ERC20WithdrawAssetCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		SetDepositMinimum: ERC20SetDepositMinimumCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		SetDepositMaximum: ERC20SetDepositMaximumCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
	}

	return erc20Cmd
}

type ERC20SetDepositMinimumCmd struct {
	Config nodewallets.Config

	TokenAddress  string `long:"token-address" required:"true" description:"The Ethereum address of the new token"`
	Amount        string `long:"amount" required:"true" description:"The amount to be withdrawn"`
	BridgeAddress string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
	Nonce         string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20SetDepositMinimumCmd) Execute(_ []string) error {
	pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := nodewallets.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	amount, overflowed := num.UintFromString(opts.Amount, 10)
	if overflowed {
		return errors.New("invalid amount, needs to be base 10")
	}

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20Logic.SetDepositMinimum(
		opts.TokenAddress, amount, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20SetDepositMaximumCmd struct {
	Config nodewallets.Config

	TokenAddress  string `long:"token-address" required:"true" description:"The Ethereum address of the new token"`
	Amount        string `long:"amount" required:"true" description:"The amount to be withdrawn"`
	BridgeAddress string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
	Nonce         string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20SetDepositMaximumCmd) Execute(_ []string) error {
	pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := nodewallets.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	amount, overflowed := num.UintFromString(opts.Amount, 10)
	if overflowed {
		return errors.New("invalid amount, needs to be base 10")
	}

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20Logic.SetDepositMaximum(
		opts.TokenAddress, amount, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20WithdrawAssetCmd struct {
	Config nodewallets.Config

	TokenAddress    string `long:"token-address" required:"true" description:"The Ethereum address of the new token"`
	Amount          string `long:"amount" required:"true" description:"The amount to be withdrawn"`
	ReceiverAddress string `long:"receiver-address" required:"true" description:"The ethereum address of the wallet which is to receive the funds"`
	BridgeAddress   string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
	Nonce           string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20WithdrawAssetCmd) Execute(_ []string) error {
	pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := nodewallets.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	amount, overflowed := num.UintFromString(opts.Amount, 10)
	if overflowed {
		return errors.New("invalid amount, needs to be base 10")
	}

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20Logic.WithdrawAsset(
		opts.TokenAddress, amount, opts.ReceiverAddress, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20ListAssetCmd struct {
	Config nodewallets.Config

	TokenAddress  string `long:"token-address" required:"true" description:"The Ethereum address of the new token"`
	VegaAssetID   string `long:"vega-asset-id" required:"true" description:"The vega ID for this new token"`
	BridgeAddress string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
	Nonce         string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20ListAssetCmd) Execute(_ []string) error {
	pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := nodewallets.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20Logic.ListAsset(
		opts.TokenAddress, opts.VegaAssetID, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20RemoveAssetCmd struct {
	Config nodewallets.Config

	TokenAddress  string `long:"token-address" required:"true" description:"The Ethereum address of the new token"`
	BridgeAddress string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
	Nonce         string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20RemoveAssetCmd) Execute(_ []string) error {
	pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := nodewallets.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20Logic.RemoveAsset(
		opts.TokenAddress, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20AddSignerCmd struct {
	Config nodewallets.Config

	NewSigner string `long:"new-signer" required:"true" description:"Ethereum address of the new signer"`
	Submitter string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce     string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20AddSignerCmd) Execute(_ []string) error {
	pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := nodewallets.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	multiSigControl := bridges.NewERC20MultiSigControl(w)
	bundle, err := multiSigControl.AddSigner(
		opts.NewSigner, opts.Submitter, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20RemoveSignerCmd struct {
	Config    nodewallets.Config
	OldSigner string `long:"old-signer" required:"true" description:"Ethereum address of signer to remove"`
	Submitter string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce     string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20RemoveSignerCmd) Execute(_ []string) error {
	pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := nodewallets.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	multiSigControl := bridges.NewERC20MultiSigControl(w)
	bundle, err := multiSigControl.RemoveSigner(
		opts.OldSigner, opts.Submitter, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20SetThresholdCmd struct {
	Config       nodewallets.Config
	NewThreshold uint16 `long:"new-threshold" required:"true" description:"The new threshold to be used on the bridge"`
	Submitter    string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce        string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20SetThresholdCmd) Execute(_ []string) error {
	if opts.NewThreshold == 0 || opts.NewThreshold > 1000 {
		return fmt.Errorf("invalid new threshold, required to be > 0 and <= 1000, got %d", opts.NewThreshold)
	}

	pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(erc20Cmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := nodewallets.GetEthereumWallet(opts.Config.ETH, vegaPaths, pass)
	if err != nil {
		return fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
	}

	multiSigControl := bridges.NewERC20MultiSigControl(w)
	bundle, err := multiSigControl.SetThreshold(
		opts.NewThreshold, opts.Submitter, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}
