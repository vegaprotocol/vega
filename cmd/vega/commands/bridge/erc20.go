// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package bridge

import (
	"errors"
	"fmt"
	"sort"
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
	PrivateKey string `description:"A ethereum private key to be use to sign the messages"                                         long:"private-key" required:"false"`
	ChainID    string `description:"The chain-id of the EVM bridge. Not required if generating signatures for the Ethereum bridge" long:"chain-id"    required:"false"`

	AddSigner              ERC20AddSignerCmd              `command:"add_signer"                description:"Create signature to add a new signer to the erc20 bridge"`
	RemoveSigner           ERC20RemoveSignerCmd           `command:"remove_signer"             description:"Create signature to remove a signer from the erc20 bridge"`
	VerifyRemoveSigner     ERC20VerifyRemoveSignerCmd     `command:"verify_remove_signer"      description:"Verify signatures to remove a signer from the erc20 bridge"`
	SetThreshold           ERC20SetThresholdCmd           `command:"set_threshold"             description:"Create signature to change the threshold of required signature to apply changes to the bridge"`
	BurnNonce              ERC20BurnNonceCmd              `command:"burn_nonce"                description:"Create signature to burn and existing nonce in order to prevent it to be used on the bridge"`
	ListAsset              ERC20ListAssetCmd              `command:"list_asset"                description:"Add a new erc20 asset to the erc20 bridge"`
	VerifyListAsset        ERC20VerifyListAssetCmd        `command:"verify_list_asset"         description:"Verify signatures to add a new erc20 asset to the erc20 bridge"`
	RemoveAsset            ERC20RemoveAssetCmd            `command:"remove_asset"              description:"Remove an erc20 asset from the erc20 bridge"`
	WithdrawAsset          ERC20WithdrawAssetCmd          `command:"withdraw_asset"            description:"Withdraw ERC20 from the bridge"`
	VerifyWithdrawAsset    ERC20VerifyWithdrawAssetCmd    `command:"verify_withdraw_asset"     description:"Verify withdraw ERC20 from the bridge"`
	SetBridgeAddress       ERC20SetBridgeAddressCmd       `command:"set_bridge_address"        description:"Update the bridge address use by the asset pool"`
	SetMultisigControl     ERC20SetMultisigControlCmd     `command:"set_multisig_control"      description:"Update the bridge address use by the asset pool"`
	VerifyGlobalResume     ERC20VerifyGlobalResumeCmd     `command:"verify_global_resume"      description:"Verify the signature to resume usage of the bridge"`
	GlobalResume           ERC20GlobalResumeCmd           `command:"global_resume"             description:"Build the signature to resume usage of the bridge"`
	GlobalStop             ERC20GlobalStopCmd             `command:"global_stop"               description:"Build the signature to stop the bridge"`
	SetWithdrawDelay       ERC20SetWithdrawDelayCmd       `command:"set_withdraw_delay"        description:"Update the withdraw delay for all asset"`
	VerifySetWithdrawDelay ERC20VerifySetWithdrawDelayCmd `command:"verify_set_withdraw_delay" description:"Verify signatures to update the withdraw delay for all asset"`
	SetAssetLimits         ERC20SetAssetLimitsCmd         `command:"set_asset_limits"          description:"Update the limits for an asset"`
	VerifySetAssetLimits   ERC20VerifySetAssetLimitsCmd   `command:"verify_set_asset_limits"   description:"Verify signatures to update the limits for an asset"`
}

var erc20Cmd *ERC20Cmd

func (e *ERC20Cmd) GetSigner() (bridges.Signer, error) {
	if len(e.PrivateKey) <= 0 {
		pass, err := erc20Cmd.PassphraseFile.Get("node wallet", false)
		if err != nil {
			return nil, err
		}

		vegaPaths := paths.New(e.VegaHome)

		if _, _, err := config.EnsureNodeConfig(vegaPaths); err != nil {
			return nil, err
		}

		s, err := nodewallets.GetEthereumWallet(vegaPaths, pass)
		if err != nil {
			return nil, fmt.Errorf("couldn't get Ethereum node wallet: %w", err)
		}

		return s, nil
	}

	s, err := NewPrivKeySigner(e.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("couldn't load private key: %w", err)
	}

	return s, nil
}

func ERC20() *ERC20Cmd {
	erc20Cmd = &ERC20Cmd{
		AddSigner:              ERC20AddSignerCmd{},
		RemoveSigner:           ERC20RemoveSignerCmd{},
		VerifyRemoveSigner:     ERC20VerifyRemoveSignerCmd{},
		SetThreshold:           ERC20SetThresholdCmd{},
		ListAsset:              ERC20ListAssetCmd{},
		VerifyListAsset:        ERC20VerifyListAssetCmd{},
		RemoveAsset:            ERC20RemoveAssetCmd{},
		WithdrawAsset:          ERC20WithdrawAssetCmd{},
		VerifyWithdrawAsset:    ERC20VerifyWithdrawAssetCmd{},
		SetAssetLimits:         ERC20SetAssetLimitsCmd{},
		VerifySetAssetLimits:   ERC20VerifySetAssetLimitsCmd{},
		SetBridgeAddress:       ERC20SetBridgeAddressCmd{},
		SetMultisigControl:     ERC20SetMultisigControlCmd{},
		VerifyGlobalResume:     ERC20VerifyGlobalResumeCmd{},
		GlobalResume:           ERC20GlobalResumeCmd{},
		GlobalStop:             ERC20GlobalStopCmd{},
		SetWithdrawDelay:       ERC20SetWithdrawDelayCmd{},
		VerifySetWithdrawDelay: ERC20VerifySetWithdrawDelayCmd{},
		BurnNonce:              ERC20BurnNonceCmd{},
	}
	return erc20Cmd
}

type ERC20WithdrawAssetCmd struct {
	TokenAddress    string `description:"The Ethereum address of the new token"                                long:"token-address"    required:"true"`
	Amount          string `description:"The amount to be withdrawn"                                           long:"amount"           required:"true"`
	ReceiverAddress string `description:"The ethereum address of the wallet which is to receive the funds"     long:"receiver-address" required:"true"`
	BridgeAddress   string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address"   required:"true"`
	Nonce           string `description:"A nonce for this signature"                                           long:"nonce"            required:"true"`
	Creation        int64  `description:"creation time of the withdrawal (timestamp)"                          long:"creation"         required:"true"`
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

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	bundle, err := erc20Logic.WithdrawAsset(
		opts.TokenAddress, amount, opts.ReceiverAddress, creation, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20VerifyWithdrawAssetCmd struct {
	TokenAddress    string `description:"The Ethereum address of the new token"                                long:"token-address"    required:"true"`
	Amount          string `description:"The amount to be withdrawn"                                           long:"amount"           required:"true"`
	ReceiverAddress string `description:"The ethereum address of the wallet which is to receive the funds"     long:"receiver-address" required:"true"`
	BridgeAddress   string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address"   required:"true"`
	Nonce           string `description:"A nonce for this signature"                                           long:"nonce"            required:"true"`
	Creation        int64  `description:"creation time of the withdrawal (timestamp)"                          long:"creation"         required:"true"`
	Signatures      string `description:"signatures of the withdrawal"                                         long:"signatures"       required:"true"`
}

func (opts *ERC20VerifyWithdrawAssetCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
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

	if len(opts.Signatures) <= 0 {
		return errors.New("missing signatures")
	}

	if (len(opts.Signatures)-2)%130 != 0 {
		return errors.New("invalid signatures format")
	}

	erc20Logic := bridges.NewERC20Logic(nil, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	addresses, err := erc20Logic.VerifyWithdrawAsset(
		opts.TokenAddress, amount, opts.ReceiverAddress, creation, nonce, opts.Signatures,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	sort.Strings(addresses)
	for _, v := range addresses {
		fmt.Printf("%v\n", v)
	}
	return nil
}

type ERC20ListAssetCmd struct {
	TokenAddress      string `description:"The Ethereum address of the new token"                                long:"token-address"      required:"true"`
	VegaAssetID       string `description:"The vega ID for this new token"                                       long:"vega-asset-id"      required:"true"`
	BridgeAddress     string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address"     required:"true"`
	Nonce             string `description:"A nonce for this signature"                                           long:"nonce"              required:"true"`
	LifetimeLimit     string `description:"The lifetime deposit limit for the asset"                             long:"lifetime-limit"     required:"true"`
	WithdrawThreshold string `description:"The withdrawal threshold for this asset"                              long:"withdraw-threshold" required:"true"`
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

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	bundle, err := erc20Logic.ListAsset(
		opts.TokenAddress, opts.VegaAssetID, lifetimeLimit, withdrawThreshod, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20VerifyListAssetCmd struct {
	TokenAddress      string `description:"The Ethereum address of the new token"                                long:"token-address"      required:"true"`
	VegaAssetID       string `description:"The vega ID for this new token"                                       long:"vega-asset-id"      required:"true"`
	BridgeAddress     string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address"     required:"true"`
	Nonce             string `description:"A nonce for this signature"                                           long:"nonce"              required:"true"`
	LifetimeLimit     string `description:"The lifetime deposit limit for the asset"                             long:"lifetime-limit"     required:"true"`
	WithdrawThreshold string `description:"The withdrawal threshold for this asset"                              long:"withdraw-threshold" required:"true"`
	Signatures        string `description:"The signature bundle to verify"                                       long:"signatures"         required:"true"`
}

func (opts *ERC20VerifyListAssetCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
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

	if len(opts.Signatures) <= 0 {
		return errors.New("missing signatures")
	}

	if (len(opts.Signatures)-2)%130 != 0 {
		return errors.New("invalid signatures format")
	}

	erc20Logic := bridges.NewERC20Logic(nil, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	addresses, err := erc20Logic.VerifyListAsset(
		opts.TokenAddress, opts.VegaAssetID, lifetimeLimit, withdrawThreshod, nonce, opts.Signatures,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	sort.Strings(addresses)
	for _, v := range addresses {
		fmt.Printf("%v\n", v)
	}
	return nil
}

type ERC20RemoveAssetCmd struct {
	TokenAddress  string `description:"The Ethereum address of the new token"                                long:"token-address"  required:"true"`
	BridgeAddress string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address" required:"true"`
	Nonce         string `description:"A nonce for this signature"                                           long:"nonce"          required:"true"`
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

	erc20Logic := bridges.NewERC20Logic(w, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
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
	NewSigner string `description:"Ethereum address of the new signer"                   long:"new-signer" required:"true"`
	Submitter string `description:"Ethereum address of the submitter of the transaction" long:"submitter"  required:"true"`
	Nonce     string `description:"A nonce for this signature"                           long:"nonce"      required:"true"`
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

	multiSigControl := bridges.NewERC20MultiSigControl(w, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
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
	OldSigner string `description:"Ethereum address of signer to remove"                 long:"old-signer" required:"true"`
	Submitter string `description:"Ethereum address of the submitter of the transaction" long:"submitter"  required:"true"`
	Nonce     string `description:"A nonce for this signature"                           long:"nonce"      required:"true"`
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

	multiSigControl := bridges.NewERC20MultiSigControl(w, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	bundle, err := multiSigControl.RemoveSigner(
		opts.OldSigner, opts.Submitter, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20VerifyRemoveSignerCmd struct {
	OldSigner  string `description:"Ethereum address of signer to remove"                 long:"old-signer" required:"true"`
	Submitter  string `description:"Ethereum address of the submitter of the transaction" long:"submitter"  required:"true"`
	Nonce      string `description:"A nonce for this signature"                           long:"nonce"      required:"true"`
	Signatures string `description:"The list of signatures from the validators"           long:"signatures" required:"true"`
}

func (opts *ERC20VerifyRemoveSignerCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
	}

	if len(opts.Signatures) <= 0 {
		return errors.New("missing signatures")
	}

	if (len(opts.Signatures)-2)%130 != 0 {
		return errors.New("invalid signatures format")
	}

	multiSigControl := bridges.NewERC20MultiSigControl(nil, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	signers, err := multiSigControl.VerifyRemoveSigner(
		opts.OldSigner, opts.Submitter, nonce, opts.Signatures,
	)
	if err != nil {
		return fmt.Errorf("unable to verify signature: %w", err)
	}

	sort.Strings(signers)
	for _, v := range signers {
		fmt.Printf("%v\n", v)
	}
	return nil
}

type ERC20SetThresholdCmd struct {
	NewThreshold uint16 `description:"The new threshold to be used on the bridge"           long:"new-threshold" required:"true"`
	Submitter    string `description:"Ethereum address of the submitter of the transaction" long:"submitter"     required:"true"`
	Nonce        string `description:"A nonce for this signature"                           long:"nonce"         required:"true"`
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

	multiSigControl := bridges.NewERC20MultiSigControl(w, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
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
	Submitter string `description:"Ethereum address of the submitter of the transaction" long:"submitter" required:"true"`
	Nonce     string `description:"A nonce for this signature"                           long:"nonce"     required:"true"`
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

	multiSigControl := bridges.NewERC20MultiSigControl(w, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	bundle, err := multiSigControl.BurnNonce(opts.Submitter, nonce)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20SetBridgeAddressCmd struct {
	NewAddress       string `description:"The Ethereum address of the bridge"                                       long:"new-address"        required:"true"`
	AssetPoolAddress string `description:"The address of the vega asset pool this transaction will be submitted to" long:"asset-pool-address" required:"true"`
	Nonce            string `description:"A nonce for this signature"                                               long:"nonce"              required:"true"`
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

	erc20Logic := bridges.NewERC20AssetPool(w, opts.AssetPoolAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
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
	NewAddress       string `description:"The Ethereum address of the bridge"                                       long:"new-address"        required:"true"`
	AssetPoolAddress string `description:"The address of the vega asset pool this transaction will be submitted to" long:"asset-pool-address" required:"true"`
	Nonce            string `description:"A nonce for this signature"                                               long:"nonce"              required:"true"`
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

	erc20Logic := bridges.NewERC20AssetPool(w, opts.AssetPoolAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
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
	Nonce         string `description:"A nonce for this signature"                                           long:"nonce"          required:"true"`
	BridgeAddress string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address" required:"true"`
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

	erc20 := bridges.NewERC20Logic(w, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
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
	Nonce         string `description:"A nonce for this signature"                                           long:"nonce"          required:"true"`
	BridgeAddress string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address" required:"true"`
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

	erc20 := bridges.NewERC20Logic(w, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	bundle, err := erc20.GlobalResume(
		nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20VerifyGlobalResumeCmd struct {
	Nonce         string `description:"A nonce for this signature"                                           long:"nonce"          required:"true"`
	BridgeAddress string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address" required:"true"`
	Signatures    string `description:"The list of signatures from the validators"                           long:"signatures"     required:"true"`
}

func (opts *ERC20VerifyGlobalResumeCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
	}

	if len(opts.Signatures) <= 0 {
		return errors.New("missing signatures")
	}

	if (len(opts.Signatures)-2)%130 != 0 {
		return errors.New("invalid signatures format")
	}

	erc20 := bridges.NewERC20Logic(nil, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	signers, err := erc20.VerifyGlobalResume(
		nonce, opts.Signatures,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	sort.Strings(signers)
	for _, v := range signers {
		fmt.Printf("%v\n", v)
	}
	return nil
}

type ERC20SetAssetLimitsCmd struct {
	WithdrawThreshold      string `description:"The threshold"                                                        long:"withdraw-threshold"       required:"true"`
	DepositLifetimeMaximum string `description:"The maxium deposit allowed per address"                               long:"deposit-lifetime-maximum" required:"true"`
	Nonce                  string `description:"A nonce for this signature"                                           long:"nonce"                    required:"true"`
	BridgeAddress          string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address"           required:"true"`
	TokenAddress           string `description:"The address of the token to be used"                                  long:"token-address"            required:"true"`
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

	erc20 := bridges.NewERC20Logic(w, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	bundle, err := erc20.SetAssetLimits(
		opts.TokenAddress, depositLifetime, threshold, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20VerifySetAssetLimitsCmd struct {
	WithdrawThreshold      string `description:"The threshold"                                                        long:"withdraw-threshold"       required:"true"`
	DepositLifetimeMaximum string `description:"The maxium deposit allowed per address"                               long:"deposit-lifetime-maximum" required:"true"`
	Nonce                  string `description:"A nonce for this signature"                                           long:"nonce"                    required:"true"`
	BridgeAddress          string `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address"           required:"true"`
	TokenAddress           string `description:"The address of the token to be used"                                  long:"token-address"            required:"true"`
	Signatures             string `description:"The list of signatures from the validators"                           long:"signatures"               required:"true"`
}

func (opts *ERC20VerifySetAssetLimitsCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
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

	if len(opts.Signatures) <= 0 {
		return errors.New("missing signatures")
	}

	if (len(opts.Signatures)-2)%130 != 0 {
		return errors.New("invalid signatures format")
	}

	erc20 := bridges.NewERC20Logic(nil, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	signers, err := erc20.VerifySetAssetLimits(
		opts.TokenAddress, depositLifetime, threshold, nonce, opts.Signatures,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	sort.Strings(signers)
	for _, v := range signers {
		fmt.Printf("%v\n", v)
	}
	return nil
}

type ERC20SetWithdrawDelayCmd struct {
	Delay         time.Duration `description:"The delay to be applied to all withdrawals"                           long:"delay"          required:"true"`
	Nonce         string        `description:"A nonce for this signature"                                           long:"nonce"          required:"true"`
	BridgeAddress string        `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address" required:"true"`
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

	erc20 := bridges.NewERC20Logic(w, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	bundle, err := erc20.SetWithdrawDelay(
		opts.Delay, nonce,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	fmt.Printf("0x%v\n", bundle.Signature.Hex())
	return nil
}

type ERC20VerifySetWithdrawDelayCmd struct {
	Delay         time.Duration `description:"The delay to be applied to all withdrawals"                           long:"delay"          required:"true"`
	Nonce         string        `description:"A nonce for this signature"                                           long:"nonce"          required:"true"`
	BridgeAddress string        `description:"The address of the vega bridge this transaction will be submitted to" long:"bridge-address" required:"true"`
	Signatures    string        `description:"The signature bundle to verify"                                       long:"signatures"     required:"true"`
}

func (opts *ERC20VerifySetWithdrawDelayCmd) Execute(_ []string) error {
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	nonce, overflowed := num.UintFromString(opts.Nonce, 10)
	if overflowed {
		return errors.New("invalid nonce, needs to be base 10 and not overflow")
	}

	if len(opts.Signatures) <= 0 {
		return errors.New("missing signatures")
	}

	if (len(opts.Signatures)-2)%130 != 0 {
		return errors.New("invalid signatures format")
	}

	erc20Logic := bridges.NewERC20Logic(nil, opts.BridgeAddress, erc20Cmd.ChainID, erc20Cmd.ChainID == "")
	addresses, err := erc20Logic.VerifyWithdrawDelay(
		opts.Delay, nonce, opts.Signatures,
	)
	if err != nil {
		return fmt.Errorf("unable to generate signature: %w", err)
	}

	sort.Strings(addresses)
	for _, v := range addresses {
		fmt.Printf("%v\n", v)
	}
	return nil
}
