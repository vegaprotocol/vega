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

package types

import (
	checkpointpb "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type PayloadBankingPrimaryBridgeState struct {
	BankingBridgeState *BankingBridgeState
}

func (p PayloadBankingPrimaryBridgeState) IntoProto() *snapshot.Payload_BankingPrimaryBridgeState {
	return &snapshot.Payload_BankingPrimaryBridgeState{
		BankingPrimaryBridgeState: &snapshot.BankingBridgeState{
			BridgeState: &checkpointpb.BridgeState{
				Active:      p.BankingBridgeState.Active,
				BlockHeight: p.BankingBridgeState.BlockHeight,
				LogIndex:    p.BankingBridgeState.LogIndex,
				ChainId:     p.BankingBridgeState.ChainID,
			},
		},
	}
}

func (*PayloadBankingPrimaryBridgeState) isPayload() {}

func (p *PayloadBankingPrimaryBridgeState) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingPrimaryBridgeState) Key() string {
	return "bridgeState"
}

func (*PayloadBankingPrimaryBridgeState) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingPrimaryBridgeStateFromProto(pbbs *snapshot.Payload_BankingPrimaryBridgeState) *PayloadBankingPrimaryBridgeState {
	return &PayloadBankingPrimaryBridgeState{
		BankingBridgeState: &BankingBridgeState{
			Active:      pbbs.BankingPrimaryBridgeState.BridgeState.Active,
			BlockHeight: pbbs.BankingPrimaryBridgeState.BridgeState.BlockHeight,
			LogIndex:    pbbs.BankingPrimaryBridgeState.BridgeState.LogIndex,
			ChainID:     pbbs.BankingPrimaryBridgeState.BridgeState.ChainId,
		},
	}
}

type PayloadBankingEVMBridgeStates struct {
	BankingBridgeStates []*checkpointpb.BridgeState
}

func (p PayloadBankingEVMBridgeStates) IntoProto() *snapshot.Payload_BankingEvmBridgeStates {
	return &snapshot.Payload_BankingEvmBridgeStates{
		BankingEvmBridgeStates: &snapshot.BankingEVMBridgeStates{
			BridgeStates: p.BankingBridgeStates,
		},
	}
}

func (*PayloadBankingEVMBridgeStates) isPayload() {}

func (p *PayloadBankingEVMBridgeStates) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingEVMBridgeStates) Key() string {
	return "evmBridgeStates"
}

func (*PayloadBankingEVMBridgeStates) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingEVMBridgeStatesFromProto(pbbs *snapshot.Payload_BankingEvmBridgeStates) *PayloadBankingEVMBridgeStates {
	return &PayloadBankingEVMBridgeStates{
		BankingBridgeStates: pbbs.BankingEvmBridgeStates.BridgeStates,
	}
}

type BankingBridgeState struct {
	Active      bool
	BlockHeight uint64
	LogIndex    uint64
	ChainID     string
}

type PayloadBankingWithdrawals struct {
	BankingWithdrawals *BankingWithdrawals
}

func PayloadBankingWithdrawalsFromProto(pbw *snapshot.Payload_BankingWithdrawals) *PayloadBankingWithdrawals {
	return &PayloadBankingWithdrawals{
		BankingWithdrawals: BankingWithdrawalsFromProto(pbw.BankingWithdrawals),
	}
}

func (p PayloadBankingWithdrawals) IntoProto() *snapshot.Payload_BankingWithdrawals {
	return &snapshot.Payload_BankingWithdrawals{
		BankingWithdrawals: p.BankingWithdrawals.IntoProto(),
	}
}

func (*PayloadBankingWithdrawals) isPayload() {}

func (p *PayloadBankingWithdrawals) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingWithdrawals) Key() string {
	return "withdrawals"
}

func (*PayloadBankingWithdrawals) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

type BankingWithdrawals struct {
	Withdrawals []*RWithdrawal
}

func (b BankingWithdrawals) IntoProto() *snapshot.BankingWithdrawals {
	ret := snapshot.BankingWithdrawals{
		Withdrawals: make([]*snapshot.Withdrawal, 0, len(b.Withdrawals)),
	}
	for _, w := range b.Withdrawals {
		ret.Withdrawals = append(ret.Withdrawals, w.IntoProto())
	}
	return &ret
}

func BankingWithdrawalsFromProto(bw *snapshot.BankingWithdrawals) *BankingWithdrawals {
	ret := &BankingWithdrawals{
		Withdrawals: make([]*RWithdrawal, 0, len(bw.Withdrawals)),
	}
	for _, w := range bw.Withdrawals {
		ret.Withdrawals = append(ret.Withdrawals, RWithdrawalFromProto(w))
	}
	return ret
}

type RWithdrawal struct {
	Ref        string
	Withdrawal *Withdrawal
}

func (r RWithdrawal) IntoProto() *snapshot.Withdrawal {
	return &snapshot.Withdrawal{
		Ref:        r.Ref,
		Withdrawal: r.Withdrawal.IntoProto(),
	}
}

func RWithdrawalFromProto(rw *snapshot.Withdrawal) *RWithdrawal {
	return &RWithdrawal{
		Ref:        rw.Ref,
		Withdrawal: WithdrawalFromProto(rw.Withdrawal),
	}
}

type PayloadBankingDeposits struct {
	BankingDeposits *BankingDeposits
}

func (p PayloadBankingDeposits) IntoProto() *snapshot.Payload_BankingDeposits {
	return &snapshot.Payload_BankingDeposits{
		BankingDeposits: p.BankingDeposits.IntoProto(),
	}
}

func (*PayloadBankingDeposits) isPayload() {}

func (p *PayloadBankingDeposits) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingDeposits) Key() string {
	return "deposits"
}

func (*PayloadBankingDeposits) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingDepositsFromProto(pbd *snapshot.Payload_BankingDeposits) *PayloadBankingDeposits {
	return &PayloadBankingDeposits{
		BankingDeposits: BankingDepositsFromProto(pbd.BankingDeposits),
	}
}

type BankingDeposits struct {
	Deposit []*BDeposit
}

func (b BankingDeposits) IntoProto() *snapshot.BankingDeposits {
	ret := snapshot.BankingDeposits{
		Deposit: make([]*snapshot.Deposit, 0, len(b.Deposit)),
	}
	for _, d := range b.Deposit {
		ret.Deposit = append(ret.Deposit, d.IntoProto())
	}
	return &ret
}

func BankingDepositsFromProto(bd *snapshot.BankingDeposits) *BankingDeposits {
	ret := &BankingDeposits{
		Deposit: make([]*BDeposit, 0, len(bd.Deposit)),
	}
	for _, d := range bd.Deposit {
		ret.Deposit = append(ret.Deposit, BDepositFromProto(d))
	}
	return ret
}

type BDeposit struct {
	ID      string
	Deposit *Deposit
}

func (b BDeposit) IntoProto() *snapshot.Deposit {
	return &snapshot.Deposit{
		Id:      b.ID,
		Deposit: b.Deposit.IntoProto(),
	}
}

func BDepositFromProto(d *snapshot.Deposit) *BDeposit {
	return &BDeposit{
		ID:      d.Id,
		Deposit: DepositFromProto(d.Deposit),
	}
}

type PayloadBankingSeen struct {
	BankingSeen *BankingSeen
}

func (p PayloadBankingSeen) IntoProto() *snapshot.Payload_BankingSeen {
	return &snapshot.Payload_BankingSeen{
		BankingSeen: p.BankingSeen.IntoProto(),
	}
}

func (*PayloadBankingSeen) isPayload() {}

func (p *PayloadBankingSeen) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingSeen) Key() string {
	return "seen"
}

func (*PayloadBankingSeen) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingSeenFromProto(pbs *snapshot.Payload_BankingSeen) *PayloadBankingSeen {
	return &PayloadBankingSeen{
		BankingSeen: BankingSeenFromProto(pbs.BankingSeen),
	}
}

type BankingSeen struct {
	Refs                      []string
	LastSeenPrimaryEthBlock   uint64
	LastSeenSecondaryEthBlock uint64
}

func (b BankingSeen) IntoProto() *snapshot.BankingSeen {
	ret := snapshot.BankingSeen{
		Refs:                      b.Refs,
		LastSeenPrimaryEthBlock:   b.LastSeenPrimaryEthBlock,
		LastSeenSecondaryEthBlock: b.LastSeenSecondaryEthBlock,
	}
	return &ret
}

func BankingSeenFromProto(bs *snapshot.BankingSeen) *BankingSeen {
	ret := BankingSeen{
		Refs:                      bs.Refs,
		LastSeenPrimaryEthBlock:   bs.LastSeenPrimaryEthBlock,
		LastSeenSecondaryEthBlock: bs.LastSeenSecondaryEthBlock,
	}
	return &ret
}

type PayloadBankingAssetActions struct {
	BankingAssetActions *BankingAssetActions
}

func (p PayloadBankingAssetActions) IntoProto() *snapshot.Payload_BankingAssetActions {
	return &snapshot.Payload_BankingAssetActions{
		BankingAssetActions: p.BankingAssetActions.IntoProto(),
	}
}

func (*PayloadBankingAssetActions) isPayload() {}

func (p *PayloadBankingAssetActions) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingAssetActions) Key() string {
	return "assetActions"
}

func (*PayloadBankingAssetActions) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingAssetActionsFromProto(pbs *snapshot.Payload_BankingAssetActions) *PayloadBankingAssetActions {
	return &PayloadBankingAssetActions{
		BankingAssetActions: BankingAssetActionsFromProto(pbs.BankingAssetActions),
	}
}

type BankingAssetActions struct {
	AssetAction []*AssetAction
}

func (a *BankingAssetActions) IntoProto() *snapshot.BankingAssetActions {
	ret := snapshot.BankingAssetActions{
		AssetAction: make([]*checkpointpb.AssetAction, 0, len(a.AssetAction)),
	}
	for _, aa := range a.AssetAction {
		ret.AssetAction = append(ret.AssetAction, aa.IntoProto())
	}
	return &ret
}

func BankingAssetActionsFromProto(aa *snapshot.BankingAssetActions) *BankingAssetActions {
	ret := BankingAssetActions{
		AssetAction: make([]*AssetAction, 0, len(aa.AssetAction)),
	}

	for _, a := range aa.AssetAction {
		ret.AssetAction = append(ret.AssetAction, AssetActionFromProto(a))
	}
	return &ret
}

type AssetAction struct {
	ID                      string
	State                   uint32
	Asset                   string
	BlockNumber             uint64
	TxIndex                 uint64
	Hash                    string
	ChainID                 string
	BuiltinD                *BuiltinAssetDeposit
	Erc20D                  *ERC20Deposit
	Erc20AL                 *ERC20AssetList
	ERC20AssetLimitsUpdated *ERC20AssetLimitsUpdated
	BridgeStopped           bool
	BridgeResume            bool
}

func (aa *AssetAction) IntoProto() *checkpointpb.AssetAction {
	ret := &checkpointpb.AssetAction{
		Id:                 aa.ID,
		State:              aa.State,
		Asset:              aa.Asset,
		BlockNumber:        aa.BlockNumber,
		TxIndex:            aa.TxIndex,
		Hash:               aa.Hash,
		Erc20BridgeStopped: aa.BridgeStopped,
		Erc20BridgeResumed: aa.BridgeResume,
		ChainId:            aa.ChainID,
	}
	if aa.BuiltinD != nil {
		ret.BuiltinDeposit = aa.BuiltinD.IntoProto()
	}
	if aa.Erc20D != nil {
		ret.Erc20Deposit = aa.Erc20D.IntoProto()
	}
	if aa.Erc20AL != nil {
		ret.AssetList = aa.Erc20AL.IntoProto()
	}
	if aa.ERC20AssetLimitsUpdated != nil {
		ret.Erc20AssetLimitsUpdated = aa.ERC20AssetLimitsUpdated.IntoProto()
	}
	return ret
}

func AssetActionFromProto(a *checkpointpb.AssetAction) *AssetAction {
	aa := &AssetAction{
		ID:            a.Id,
		State:         a.State,
		Asset:         a.Asset,
		BlockNumber:   a.BlockNumber,
		ChainID:       a.ChainId,
		TxIndex:       a.TxIndex,
		Hash:          a.Hash,
		BridgeStopped: a.Erc20BridgeStopped,
		BridgeResume:  a.Erc20BridgeResumed,
	}

	if a.Erc20Deposit != nil {
		erc20d, err := NewERC20DepositFromProto(a.Erc20Deposit)
		if err == nil {
			aa.Erc20D = erc20d
		}
	}

	if a.BuiltinDeposit != nil {
		builtind, err := NewBuiltinAssetDepositFromProto(a.BuiltinDeposit)
		if err == nil {
			aa.BuiltinD = builtind
		}
	}

	if a.AssetList != nil {
		aa.Erc20AL = NewERC20AssetListFromProto(a.AssetList)
	}

	if a.Erc20AssetLimitsUpdated != nil {
		aa.ERC20AssetLimitsUpdated = NewERC20AssetLimitsUpdatedFromProto(a.Erc20AssetLimitsUpdated)
	}

	return aa
}

type PayloadBankingScheduledTransfers struct {
	BankingScheduledTransfers []*checkpointpb.ScheduledTransferAtTime
}

func (p PayloadBankingScheduledTransfers) IntoProto() *snapshot.Payload_BankingScheduledTransfers {
	return &snapshot.Payload_BankingScheduledTransfers{
		BankingScheduledTransfers: &snapshot.BankingScheduledTransfers{
			TransfersAtTime: p.BankingScheduledTransfers,
		},
	}
}

func (*PayloadBankingScheduledTransfers) isPayload() {}

func (p *PayloadBankingScheduledTransfers) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingScheduledTransfers) Key() string {
	return "scheduledTransfers"
}

func (*PayloadBankingScheduledTransfers) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingScheduledTransfersFromProto(pbd *snapshot.Payload_BankingScheduledTransfers) *PayloadBankingScheduledTransfers {
	return &PayloadBankingScheduledTransfers{
		BankingScheduledTransfers: pbd.BankingScheduledTransfers.TransfersAtTime,
	}
}

type PayloadBankingRecurringGovernanceTransfers struct {
	BankingRecurringGovernanceTransfers []*checkpointpb.GovernanceTransfer
}

func (p PayloadBankingRecurringGovernanceTransfers) IntoProto() *snapshot.Payload_BankingRecurringGovernanceTransfers {
	return &snapshot.Payload_BankingRecurringGovernanceTransfers{
		BankingRecurringGovernanceTransfers: &snapshot.BankingRecurringGovernanceTransfers{
			RecurringTransfers: p.BankingRecurringGovernanceTransfers,
		},
	}
}

func (*PayloadBankingRecurringGovernanceTransfers) isPayload() {}

func (p *PayloadBankingRecurringGovernanceTransfers) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingRecurringGovernanceTransfers) Key() string {
	return "recurringGovernanceTransfers"
}

func (*PayloadBankingRecurringGovernanceTransfers) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingRecurringGovernanceTransfersFromProto(pbd *snapshot.Payload_BankingRecurringGovernanceTransfers) *PayloadBankingRecurringGovernanceTransfers {
	return &PayloadBankingRecurringGovernanceTransfers{
		BankingRecurringGovernanceTransfers: pbd.BankingRecurringGovernanceTransfers.RecurringTransfers,
	}
}

type PayloadBankingScheduledGovernanceTransfers struct {
	BankingScheduledGovernanceTransfers []*checkpointpb.ScheduledGovernanceTransferAtTime
}

func (p PayloadBankingScheduledGovernanceTransfers) IntoProto() *snapshot.Payload_BankingScheduledGovernanceTransfers {
	return &snapshot.Payload_BankingScheduledGovernanceTransfers{
		BankingScheduledGovernanceTransfers: &snapshot.BankingScheduledGovernanceTransfers{
			TransfersAtTime: p.BankingScheduledGovernanceTransfers,
		},
	}
}

func (*PayloadBankingScheduledGovernanceTransfers) isPayload() {}

func (p *PayloadBankingScheduledGovernanceTransfers) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingScheduledGovernanceTransfers) Key() string {
	return "scheduledGovernanceTransfers"
}

func (*PayloadBankingScheduledGovernanceTransfers) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingScheduledGovernanceTransfersFromProto(pbd *snapshot.Payload_BankingScheduledGovernanceTransfers) *PayloadBankingScheduledGovernanceTransfers {
	return &PayloadBankingScheduledGovernanceTransfers{
		BankingScheduledGovernanceTransfers: pbd.BankingScheduledGovernanceTransfers.TransfersAtTime,
	}
}

type PayloadBankingTransferFeeDiscounts struct {
	BankingTransferFeeDiscounts *snapshot.BankingTransferFeeDiscounts
}

func (p PayloadBankingTransferFeeDiscounts) IntoProto() *snapshot.Payload_BankingTransferFeeDiscounts {
	return &snapshot.Payload_BankingTransferFeeDiscounts{
		BankingTransferFeeDiscounts: &snapshot.BankingTransferFeeDiscounts{
			PartyAssetDiscount: p.BankingTransferFeeDiscounts.PartyAssetDiscount,
		},
	}
}

func (*PayloadBankingTransferFeeDiscounts) isPayload() {}

func (p *PayloadBankingTransferFeeDiscounts) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingTransferFeeDiscounts) Key() string {
	return "transferFeeDiscounts"
}

func (*PayloadBankingTransferFeeDiscounts) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingTransferFeeDiscountsFromProto(pbd *snapshot.Payload_BankingTransferFeeDiscounts) *PayloadBankingTransferFeeDiscounts {
	return &PayloadBankingTransferFeeDiscounts{
		BankingTransferFeeDiscounts: pbd.BankingTransferFeeDiscounts,
	}
}

type PayloadBankingRecurringTransfers struct {
	BankingRecurringTransfers *checkpointpb.RecurringTransfers
}

func (p PayloadBankingRecurringTransfers) IntoProto() *snapshot.Payload_BankingRecurringTransfers {
	return &snapshot.Payload_BankingRecurringTransfers{
		BankingRecurringTransfers: &snapshot.BankingRecurringTransfers{
			RecurringTransfers: p.BankingRecurringTransfers,
		},
	}
}

func (*PayloadBankingRecurringTransfers) isPayload() {}

func (p *PayloadBankingRecurringTransfers) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadBankingRecurringTransfers) Key() string {
	return "recurringTransfers"
}

func (*PayloadBankingRecurringTransfers) Namespace() SnapshotNamespace {
	return BankingSnapshot
}

func PayloadBankingRecurringTransfersFromProto(pbd *snapshot.Payload_BankingRecurringTransfers) *PayloadBankingRecurringTransfers {
	return &PayloadBankingRecurringTransfers{
		BankingRecurringTransfers: pbd.BankingRecurringTransfers.RecurringTransfers,
	}
}
