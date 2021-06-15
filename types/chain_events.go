//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type Deposit = proto.Deposit
type WithdrawExt = proto.WithdrawExt
type WithdrawExt_Erc20 = proto.WithdrawExt_Erc20
type BuiltinAssetDeposit = proto.BuiltinAssetDeposit
type ERC20Deposit = proto.ERC20Deposit
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
	// Default value, always invalid
	Withdrawal_STATUS_UNSPECIFIED Withdrawal_Status = 0
	// The withdrawal is open and being processed by the network
	Withdrawal_STATUS_OPEN Withdrawal_Status = 1
	// The withdrawal have been cancelled
	Withdrawal_STATUS_CANCELLED Withdrawal_Status = 2
	// The withdrawal went through and is fully finalised, the funds are removed from the
	// Vega network and are unlocked on the foreign chain bridge, for example, on the Ethereum network
	Withdrawal_STATUS_FINALIZED Withdrawal_Status = 3
)

type Deposit_Status = proto.Deposit_Status

const (
	// Default value, always invalid
	Deposit_STATUS_UNSPECIFIED Deposit_Status = 0
	// The deposit is being processed by the network
	Deposit_STATUS_OPEN Deposit_Status = 1
	// The deposit has been cancelled by the network
	Deposit_STATUS_CANCELLED Deposit_Status = 2
	// The deposit has been finalised and accounts have been updated
	Deposit_STATUS_FINALIZED Deposit_Status = 3
)

type Withdrawal struct {
	// Unique identifier for the withdrawal
	ID string
	// Unique party identifier of the user initiating the withdrawal
	PartyID string
	// The amount to be withdrawn
	Amount *num.Uint
	// The asset we want to withdraw funds from
	Asset string
	// The status of the withdrawal
	Status Withdrawal_Status
	// The reference which is used by the foreign chain
	// to refer to this withdrawal
	Ref string
	// The hash of the foreign chain for this transaction
	TxHash string
	// Timestamp for when the network started to process this withdrawal
	CreationDate int64
	// Timestamp for when the withdrawal was finalised by the network
	WithdrawalDate int64
	// The time until when the withdrawal is valid
	ExpirationDate int64
	// Foreign chain specifics
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
