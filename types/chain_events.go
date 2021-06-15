//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type WithdrawExt = proto.WithdrawExt
type WithdrawExt_Erc20 = proto.WithdrawExt_Erc20
type ERC20AssetList = proto.ERC20AssetList
type ERC20Withdrawal = proto.ERC20Withdrawal
type Erc20WithdrawExt = proto.Erc20WithdrawExt
type BuiltinAsset = proto.BuiltinAsset
type ERC20 = proto.ERC20
type ChainEvent_Builtin = commandspb.ChainEvent_Builtin
type ChainEvent_Erc20 = commandspb.ChainEvent_Erc20
type ChainEvent_Btc = commandspb.ChainEvent_Btc
type ChainEvent_Validator = commandspb.ChainEvent_Validator
type BuiltinAssetEvent_Deposit = proto.BuiltinAssetEvent_Deposit
type BuiltinAssetEvent_Withdrawal = proto.BuiltinAssetEvent_Withdrawal
type ERC20Event_AssetList = proto.ERC20Event_AssetList
type ERC20Event_AssetDelist = proto.ERC20Event_AssetDelist
type ERC20Event_Deposit = proto.ERC20Event_Deposit
type ERC20Event_Withdrawal = proto.ERC20Event_Withdrawal

type Withdrawal_Status = proto.Withdrawal_Status

const (
	// Withdrawal_STATUS_UNSPECIFIED Default value, always invalid
	Withdrawal_STATUS_UNSPECIFIED Withdrawal_Status = 0
	// Withdrawal_STATUS_OPEN The withdrawal is open and being processed by the network
	Withdrawal_STATUS_OPEN Withdrawal_Status = 1
	// Withdrawal_STATUS_CANCELLED The withdrawal have been cancelled
	Withdrawal_STATUS_CANCELLED Withdrawal_Status = 2
	// Withdrawal_STATUS_FINALIZED The withdrawal went through and is fully finalised, the funds are removed from the
	// Vega network and are unlocked on the foreign chain bridge, for example, on the Ethereum network
	Withdrawal_STATUS_FINALIZED Withdrawal_Status = 3
)

type Withdrawal struct {
	// ID Unique identifier for the withdrawal
	ID string
	// PartyID Unique party identifier of the user initiating the withdrawal
	PartyID string
	// Amount The amount to be withdrawn
	Amount *num.Uint
	// Asset The asset we want to withdraw funds from
	Asset string
	// Status The status of the withdrawal
	Status Withdrawal_Status
	// Ref The reference which is used by the foreign chain
	// to refer to this withdrawal
	Ref string
	// TxHash The hash of the foreign chain for this transaction
	TxHash string
	// CreationDate Timestamp for when the network started to process this withdrawal
	CreationDate int64
	// WithdrawalDate Timestamp for when the withdrawal was finalised by the network
	WithdrawalDate int64
	// ExpirationDate The time until when the withdrawal is valid
	ExpirationDate int64
	// Ext Foreign chain specifics
	Ext *WithdrawExt
}

func (w *Withdrawal) IntoProto() *proto.Withdrawal {
	return &proto.Withdrawal{
		Id:                 w.ID,
		PartyId:            w.PartyID,
		Amount:             w.Amount.Uint64(),
		Asset:              w.Asset,
		Status:             w.Status,
		Ref:                w.Ref,
		TxHash:             w.TxHash,
		Expiry:             w.ExpirationDate,
		CreatedTimestamp:   w.CreationDate,
		WithdrawnTimestamp: w.WithdrawalDate,
		Ext:                w.Ext,
	}
}

type Deposit_Status = proto.Deposit_Status

const (
	// Deposit_STATUS_UNSPECIFIED Default value, always invalid
	Deposit_STATUS_UNSPECIFIED Deposit_Status = 0
	// Deposit_STATUS_OPEN The deposit is being processed by the network
	Deposit_STATUS_OPEN Deposit_Status = 1
	// Deposit_STATUS_CANCELLED The deposit has been cancelled by the network
	Deposit_STATUS_CANCELLED Deposit_Status = 2
	// Deposit_STATUS_FINALIZED The deposit has been finalised and accounts have been updated
	Deposit_STATUS_FINALIZED Deposit_Status = 3
)

// Deposit represent a deposit on to the Vega network
type Deposit struct {
	// ID Unique identifier for the deposit
	ID string
	// Status of the deposit
	Status Deposit_Status
	// Party identifier of the user initiating the deposit
	PartyID string
	// Asset The Vega asset targeted by this deposit
	Asset string
	// Amount The amount to be deposited
	Amount *num.Uint
	// TxHash The hash of the transaction from the foreign chain
	TxHash string
	// Timestamp for when the Vega account was updated with the deposit
	CreditDate int64
	// Timestamp for when the deposit was created on the Vega network
	CreationDate int64
}

func (d *Deposit) IntoProto() *proto.Deposit {
	return &proto.Deposit{
		Id:                d.ID,
		Status:            d.Status,
		PartyId:           d.PartyID,
		Asset:             d.Asset,
		Amount:            d.Amount.String(),
		TxHash:            d.TxHash,
		CreditedTimestamp: d.CreditDate,
		CreatedTimestamp:  d.CreationDate,
	}
}

// BuiltinAssetDeposit represents a deposit for a Vega built-in asset
type BuiltinAssetDeposit struct {
	// VegaAssetID A Vega network internal asset identifier
	VegaAssetID string
	// PartyID A Vega party identifier (pub-key)
	PartyID string
	// Amount The amount to be deposited
	Amount *num.Uint
}

func NewBuiltinAssetDeposit(d *proto.BuiltinAssetDeposit) *BuiltinAssetDeposit {
	return &BuiltinAssetDeposit{
		VegaAssetID: d.VegaAssetId,
		PartyID:     d.PartyId,
		Amount:      num.NewUint(d.Amount),
	}
}

func (d BuiltinAssetDeposit) String() string {
	return fmt.Sprintf("VegaAssetID: %s, PartyID: %s, Amount: %s",
		d.VegaAssetID,
		d.PartyID,
		d.Amount.String(),
	)
}

// ERC20Deposit represents an asset deposit for an ERC20 token
type ERC20Deposit struct {
	// VegaAssetID The vega network internal identifier of the asset
	VegaAssetID string
	// SourceEthereumAddress The Ethereum wallet that initiated the deposit
	SourceEthereumAddress string
	// TargetPartyID The Vega party identifier (pub-key) which is the target of the deposit
	TargetPartyID string
	// Amount The amount to be deposited
	Amount *num.Uint
}

func NewERC20Deposit(d *proto.ERC20Deposit) (*ERC20Deposit, error) {
	amount, err := strconv.ParseUint(d.Amount, 10, 64)
	if err != nil {
		return nil, err
	}

	return &ERC20Deposit{
		VegaAssetID:           d.VegaAssetId,
		TargetPartyID:         d.TargetPartyId,
		Amount:                num.NewUint(amount),
		SourceEthereumAddress: d.SourceEthereumAddress,
	}, nil
}

func (d ERC20Deposit) String() string {
	return fmt.Sprintf("VegaAssetID: %s, SourceEthereumAddress: %s, TargetPartyID: %s, Amount: %s",
		d.VegaAssetID,
		d.SourceEthereumAddress,
		d.TargetPartyID,
		d.Amount.String(),
	)
}
