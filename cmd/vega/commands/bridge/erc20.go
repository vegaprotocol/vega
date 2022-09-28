// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package bridge

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/bridges"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/paths"
	"github.com/jessevdk/go-flags"
)

type ERC20Cmd struct {
	config.VegaHomeFlag
	config.PassphraseFlag
	Config     nodewallets.Config
	PrivateKey string `long:"private-key" required:"false" description:"A ethereum private key to be use to sign the messages"`

	AddSigner          ERC20AddSignerCmd          `command:"add_signer" description:"Create signature to add a new signer to the erc20 bridge"`
	RemoveSigner       ERC20RemoveSignerCmd       `command:"remove_signer" description:"Create signature to remove a signer from the erc20 bridge"`
	SetThreshold       ERC20SetThresholdCmd       `command:"set_threshold" description:"Create signature to change the threshold of required signature to apply changes to the bridge"`
	BurnNonce          ERC20BurnNonceCmd          `command:"burn_nonce" description:"Create signature to burn and existing nonce in order to prevent it to be used on the bridge"`
	ListAsset          ERC20ListAssetCmd          `command:"list_asset" description:"Add a new erc20 asset to the erc20 bridge"`
	RemoveAsset        ERC20RemoveAssetCmd        `command:"remove_asset" description:"Remove an erc20 asset from the erc20 bridge"`
	WithdrawAsset      ERC20WithdrawAssetCmd      `command:"withdraw_asset" description:"Withdraw ERC20 from the bridge"`
	SetBridgeAddress   ERC20SetBridgeAddressCmd   `command:"set_bridge_address" description:"Update the bridge address use by the asset pool"`
	SetMultisigControl ERC20SetMultisigControlCmd `command:"set_multisig_control" description:"Update the bridge address use by the asset pool"`
	GlobalResume       ERC20GlobalResumeCmd       `command:"global_resume" description:"Build the signature to resume usage of the bridge"`
	GlobalStop         ERC20GlobalStopCmd         `command:"global_stop" description:"Build the signature to stop the bridge"`
	SetWithdrawDelay   ERC20SetWithdrawDelayCmd   `command:"set_withdraw_delay" description:"Update the withdraw delay for all asset"`
	SetAssetLimits     ERC20SetAssetLimitsCmd     `command:"set_asset_limits" description:"Update the limits for an asset"`
}

var erc20Cmd *ERC20Cmd

func (e *ERC20Cmd) GetSigner() (bridges.Signer, error) {
	vegaPaths := paths.New(e.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return nil, err
	}

	e.Config = conf.NodeWallet

	var s bridges.Signer
	if len(e.PrivateKey) <= 0 {
		pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
		if err != nil {
			return nil, err
		}

		s, err = nodewallets.GetEthereumWallet(vegaPaths, pass)
		if err != nil {
			return nil, fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
		}
	} else {
		s, err = NewPrivKeySigner(e.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("couldn't load private key: %w", err)
		}
	}
	return s, nil
}

func ERC20() *ERC20Cmd {
	erc20Cmd = &ERC20Cmd{
		Config:             nodewallets.NewDefaultConfig(),
		AddSigner:          ERC20AddSignerCmd{},
		RemoveSigner:       ERC20RemoveSignerCmd{},
		SetThreshold:       ERC20SetThresholdCmd{},
		ListAsset:          ERC20ListAssetCmd{},
		RemoveAsset:        ERC20RemoveAssetCmd{},
		WithdrawAsset:      ERC20WithdrawAssetCmd{},
		SetAssetLimits:     ERC20SetAssetLimitsCmd{},
		SetBridgeAddress:   ERC20SetBridgeAddressCmd{},
		SetMultisigControl: ERC20SetMultisigControlCmd{},
		GlobalResume:       ERC20GlobalResumeCmd{},
		GlobalStop:         ERC20GlobalStopCmd{},
		SetWithdrawDelay:   ERC20SetWithdrawDelayCmd{},
		BurnNonce:          ERC20BurnNonceCmd{},
	}
	return erc20Cmd
}

type ERC20WithdrawAssetCmd struct {
	TokenAddress    string `long:"token-address" required:"true" description:"The Ethereum address of the new token"`
	Amount          string `long:"amount" required:"true" description:"The amount to be withdrawn"`
	ReceiverAddress string `long:"receiver-address" required:"true" description:"The ethereum address of the wallet which is to receive the funds"`
	BridgeAddress   string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
	Nonce           string `long:"nonce" required:"true" description:"A nonce for this signature"`
	Creation        int64  `long:"creation" required:"true" description:"creation time of the withdrawal (timestamp)"`
}

func (opts *ERC20WithdrawAssetCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	amount, overflowed := num.UintFromString(opts.Amount, 10)
	if overflowed {
		return errors.New("invalid amount, needs to be base 10")
	}

	creation := time.Unix(opts.Creation, 0)

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20Logic.WithdrawAsset(
		opts.TokenAddress, amount, opts.ReceiverAddress, creation, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20ListAssetCmd struct {
	TokenAddress      string `long:"token-address" required:"true" description:"The Ethereum address of the new token"`
	VegaAssetID       string `long:"vega-asset-id" required:"true" description:"The vega ID for this new token"`
	BridgeAddress     string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
	Nonce             string `long:"nonce" required:"true" description:"A nonce for this signature"`
	LifetimeLimit     string `long:"lifetime-limit" required:"true" description:"The lifetime deposit limit for the asset"`
	WithdrawThreshold string `long:"withdraw-threshold" required:"true" description:"The withdrawal threshold for this asset"`
}

func (opts *ERC20ListAssetCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}
	lifetimeLimit, overflowed := num.UintFromString(opts.LifetimeLimit, 10)
	if overflowed {
		return errors.New("invalid lifetime-limit, needs to be base 10")
	}
	withdrawThreshod, overflowed := num.UintFromString(opts.WithdrawThreshold, 10)
	if overflowed {
		return errors.New("invalid withdraw-threshold, needs to be base 10")
	}

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20Logic.ListAsset(
		opts.TokenAddress, opts.VegaAssetID, lifetimeLimit, withdrawThreshod, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20RemoveAssetCmd struct {
	TokenAddress  string `long:"token-address" required:"true" description:"The Ethereum address of the new token"`
	BridgeAddress string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
	Nonce         string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20RemoveAssetCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
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
	NewSigner string `long:"new-signer" required:"true" description:"Ethereum address of the new signer"`
	Submitter string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce     string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20AddSignerCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
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
	OldSigner string `long:"old-signer" required:"true" description:"Ethereum address of signer to remove"`
	Submitter string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce     string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20RemoveSignerCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
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
	NewThreshold uint16 `long:"new-threshold" required:"true" description:"The new threshold to be used on the bridge"`
	Submitter    string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce        string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20SetThresholdCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	if opts.NewThreshold == 0 || opts.NewThreshold > 1000 {
		return fmt.Errorf("invalid new threshold, required to be > 0 and <= 1000, got %d", opts.NewThreshold)
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

type ERC20BurnNonceCmd struct {
	Submitter string `long:"submitter" required:"true" description:"Ethereum address of the submitter of the transaction"`
	Nonce     string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20BurnNonceCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
	}

	multiSigControl := bridges.NewERC20MultiSigControl(w)
	bundle, err := multiSigControl.BurnNonce(opts.Submitter, nonce)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20SetBridgeAddressCmd struct {
	NewAddress       string `long:"new-address" required:"true" description:"The Ethereum address of the bridge"`
	AssetPoolAddress string `long:"asset-pool-address" required:"true" description:"The address of the vega asset pool this transaction will be submitted to"`
	Nonce            string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20SetBridgeAddressCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	erc20Logic := bridges.NewERC20AssetPool(w, opts.AssetPoolAddress)
	bundle, err := erc20Logic.SetBridgeAddress(
		opts.NewAddress, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20SetMultisigControlCmd struct {
	NewAddress       string `long:"new-address" required:"true" description:"The Ethereum address of the bridge"`
	AssetPoolAddress string `long:"asset-pool-address" required:"true" description:"The address of the vega asset pool this transaction will be submitted to"`
	Nonce            string `long:"nonce" required:"true" description:"A nonce for this signature"`
}

func (opts *ERC20SetMultisigControlCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10")
	}

	erc20Logic := bridges.NewERC20AssetPool(w, opts.AssetPoolAddress)
	bundle, err := erc20Logic.SetMultiSigControl(
		opts.NewAddress, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20GlobalStopCmd struct {
	Nonce         string `long:"nonce" required:"true" description:"A nonce for this signature"`
	BridgeAddress string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
}

func (opts *ERC20GlobalStopCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
	}

	erc20 := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20.GlobalStop(
		nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20GlobalResumeCmd struct {
	Nonce         string `long:"nonce" required:"true" description:"A nonce for this signature"`
	BridgeAddress string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
}

func (opts *ERC20GlobalResumeCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
	}

	erc20 := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20.GlobalResume(
		nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20SetAssetLimitsCmd struct {
	WithdrawThreshold      string `long:"withdraw-threshold" required:"true" description:"The threshold"`
	DepositLifetimeMaximum string `long:"deposit-lifetime-maximum" required:"true" description:"The maxium deposit allowed per address"`
	Nonce                  string `long:"nonce" required:"true" description:"A nonce for this signature"`
	BridgeAddress          string `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
	TokenAddress           string `long:"token-address" required:"true" description:"The address of the token to be used"`
}

func (opts *ERC20SetAssetLimitsCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
	}

	threshold, overflowed := num.UintFromString(opts.WithdrawThreshold, 10)
	if overflowed {
		return errors.New("invalid withdraw-threshold, needs to be base 10 and not overflow")
	}

	depositLifetime, overflowed := num.UintFromString(opts.DepositLifetimeMaximum, 10)
	if overflowed {
		return errors.New("invalid deposit-lifetime-maximum needs to be base 10 and not overflow")
	}

	erc20 := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20.SetAssetLimits(
		opts.TokenAddress, depositLifetime, threshold, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20SetWithdrawDelayCmd struct {
	Delay         time.Duration `long:"delay" required:"true" description:"The delay to be applied to all withdrawals"`
	Nonce         string        `long:"nonce" required:"true" description:"A nonce for this signature"`
	BridgeAddress string        `long:"bridge-address" required:"true" description:"The address of the vega bridge this transaction will be submitted to"`
}

func (opts *ERC20SetWithdrawDelayCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	w, err := erc20Cmd.GetSigner()
	if err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
	}

	erc20 := bridges.NewERC20Logic(w, opts.BridgeAddress)
	bundle, err := erc20.SetWithdrawDelay(
		opts.Delay, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}
