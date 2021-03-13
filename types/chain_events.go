//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import "code.vegaprotocol.io/vega/proto"

type Deposit = proto.Deposit
type Withdrawal = proto.Withdrawal
type WithdrawExt = proto.WithdrawExt
type WithdrawExt_Erc20 = proto.WithdrawExt_Erc20
type BuiltinAssetDeposit = proto.BuiltinAssetDeposit
type ERC20Deposit = proto.ERC20Deposit
type ERC20AssetList = proto.ERC20AssetList
type ERC20Withdrawal = proto.ERC20Withdrawal
type Erc20WithdrawExt = proto.Erc20WithdrawExt
type BuiltinAsset = proto.BuiltinAsset
type ERC20 = proto.ERC20

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
