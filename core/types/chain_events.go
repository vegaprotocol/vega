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

//lint:file-ignore ST1003 Ignore underscores in names, this is straight copied from the proto package to ease introducing the domain types

package types

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type WithdrawExt struct {
	Ext isWithdrawExtExt
}

func (x *WithdrawExt) String() string {
	return fmt.Sprintf(
		"ext(%s)",
		stringer.ObjToString(x.Ext),
	)
}

func (x *WithdrawExt) IntoProto() *vegapb.WithdrawExt {
	if x == nil {
		return nil
	}
	switch st := x.Ext.(type) {
	case *WithdrawExtErc20:
		return &vegapb.WithdrawExt{
			Ext: st.IntoProto(),
		}
	default:
		return nil
	}
}

func (x *WithdrawExt) GetErc20() *WithdrawExtErc20 {
	switch st := x.Ext.(type) {
	case *WithdrawExtErc20:
		return st
	default:
		return nil
	}
}

func WithdrawExtFromProto(extProto *vegapb.WithdrawExt) *WithdrawExt {
	if extProto == nil {
		return nil
	}
	var src isWithdrawExtExt
	switch st := extProto.Ext.(type) {
	case *vegapb.WithdrawExt_Erc20:
		src = WithdrawExtErc20FromProto(st)
	}
	return &WithdrawExt{
		Ext: src,
	}
}

type isWithdrawExtExt interface {
	isWithdrawExtExt()
	String() string
}

type WithdrawExtErc20 struct {
	Erc20 *Erc20WithdrawExt
}

func (x *WithdrawExtErc20) isWithdrawExtExt() {}

func (x *WithdrawExtErc20) String() string {
	return fmt.Sprintf(
		"erc20(%s)",
		stringer.PtrToString(x.Erc20),
	)
}

func (x *WithdrawExtErc20) IntoProto() *vegapb.WithdrawExt_Erc20 {
	return &vegapb.WithdrawExt_Erc20{
		Erc20: x.Erc20.IntoProto(),
	}
}

func (x *WithdrawExtErc20) GetReceiverAddress() string {
	return x.Erc20.ReceiverAddress
}

func WithdrawExtErc20FromProto(erc20 *vegapb.WithdrawExt_Erc20) *WithdrawExtErc20 {
	return &WithdrawExtErc20{
		Erc20: Erc20WithdrawExtFromProto(erc20.Erc20),
	}
}

type Erc20WithdrawExt struct {
	ReceiverAddress string
}

func (x *Erc20WithdrawExt) String() string {
	return fmt.Sprintf("receiverAddress(%s)", x.ReceiverAddress)
}

func (x *Erc20WithdrawExt) IntoProto() *vegapb.Erc20WithdrawExt {
	return &vegapb.Erc20WithdrawExt{
		ReceiverAddress: x.ReceiverAddress,
	}
}

func Erc20WithdrawExtFromProto(erc20 *vegapb.Erc20WithdrawExt) *Erc20WithdrawExt {
	return &Erc20WithdrawExt{
		ReceiverAddress: crypto.EthereumChecksumAddress(erc20.ReceiverAddress),
	}
}

type WithdrawalStatus = vegapb.Withdrawal_Status

const (
	// WithdrawalStatusUnspecified Default value, always invalid.
	WithdrawalStatusUnspecified WithdrawalStatus = 0
	// WithdrawalStatusOpen The withdrawal is open and being processed by the network.
	WithdrawalStatusOpen WithdrawalStatus = 1
	// WithdrawalStatusRejected The withdrawal have been rejected.
	WithdrawalStatusRejected WithdrawalStatus = 2
	// WithdrawalStatusFinalized The withdrawal went through and is fully finalised, the funds are removed from the
	// Vega network and are unlocked on the foreign chain bridge, for example, on the Ethereum network.
	WithdrawalStatusFinalized WithdrawalStatus = 3
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
	Status WithdrawalStatus
	// Ref The reference which is used by the foreign chain
	// to refer to this withdrawal
	Ref string
	// TxHash The hash of the foreign chain for this transaction
	TxHash string
	// CreationDate Timestamp for when the network started to process this withdrawal
	CreationDate int64
	// WithdrawalDate Timestamp for when the withdrawal was finalised by the network
	WithdrawalDate int64
	// Ext Foreign chain specifics
	Ext *WithdrawExt
}

func (w *Withdrawal) IntoProto() *vegapb.Withdrawal {
	return &vegapb.Withdrawal{
		Id:                 w.ID,
		PartyId:            w.PartyID,
		Amount:             num.UintToString(w.Amount),
		Asset:              w.Asset,
		Status:             w.Status,
		Ref:                w.Ref,
		TxHash:             w.TxHash,
		CreatedTimestamp:   w.CreationDate,
		WithdrawnTimestamp: w.WithdrawalDate,
		Ext:                w.Ext.IntoProto(),
	}
}

func WithdrawalFromProto(w *vegapb.Withdrawal) *Withdrawal {
	amt, _ := num.UintFromString(w.Amount, 10)
	return &Withdrawal{
		ID:             w.Id,
		PartyID:        w.PartyId,
		Amount:         amt,
		Asset:          w.Asset,
		Status:         w.Status,
		Ref:            w.Ref,
		TxHash:         w.TxHash,
		CreationDate:   w.CreatedTimestamp,
		WithdrawalDate: w.WithdrawnTimestamp,
		Ext:            WithdrawExtFromProto(w.Ext),
	}
}

type DepositStatus = vegapb.Deposit_Status

const (
	// DepositStatusUnspecified Default value, always invalid.
	DepositStatusUnspecified DepositStatus = 0
	// DepositStatusOpen The deposit is being processed by the network.
	DepositStatusOpen DepositStatus = 1
	// DepositStatusCancelled The deposit has been cancelled by the network.
	DepositStatusCancelled DepositStatus = 2
	// DepositStatusFinalized The deposit has been finalised and accounts have been updated.
	DepositStatusFinalized DepositStatus = 3
)

// Deposit represent a deposit on to the Vega network.
type Deposit struct {
	// ID Unique identifier for the deposit
	ID string
	// Status of the deposit
	Status DepositStatus
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

func (d *Deposit) IntoProto() *vegapb.Deposit {
	return &vegapb.Deposit{
		Id:                d.ID,
		Status:            d.Status,
		PartyId:           d.PartyID,
		Asset:             d.Asset,
		Amount:            num.UintToString(d.Amount),
		TxHash:            d.TxHash,
		CreditedTimestamp: d.CreditDate,
		CreatedTimestamp:  d.CreationDate,
	}
}

func (d *Deposit) String() string {
	return fmt.Sprintf(
		"ID(%s) status(%s) partyID(%s) asset(%s) amount(%s) txHash(%s) creditDate(%v) creationDate(%v)",
		d.ID,
		d.Status.String(),
		d.PartyID,
		d.Asset,
		stringer.PtrToString(d.Amount),
		d.TxHash,
		d.CreditDate,
		d.CreationDate,
	)
}

func DepositFromProto(d *vegapb.Deposit) *Deposit {
	amt, _ := num.UintFromString(d.Amount, 10)
	return &Deposit{
		ID:           d.Id,
		Status:       d.Status,
		PartyID:      d.PartyId,
		Asset:        d.Asset,
		Amount:       amt,
		TxHash:       d.TxHash,
		CreditDate:   d.CreditedTimestamp,
		CreationDate: d.CreatedTimestamp,
	}
}

type ChainEventERC20 struct {
	ERC20 *ERC20Event
}

func NewChainEventERC20FromProto(p *commandspb.ChainEvent_Erc20) (*ChainEventERC20, error) {
	c := ChainEventERC20{}
	var err error
	c.ERC20, err = NewERC20Event(p.Erc20)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (c ChainEventERC20) IntoProto() *commandspb.ChainEvent_Erc20 {
	p := &commandspb.ChainEvent_Erc20{
		Erc20: c.ERC20.IntoProto(),
	}
	return p
}

func (c ChainEventERC20) String() string {
	return fmt.Sprintf(
		"erc20(%s)",
		stringer.PtrToString(c.ERC20),
	)
}

type BuiltinAssetDeposit struct {
	// A Vega network internal asset identifier
	VegaAssetID string
	// A Vega party identifier (pub-key)
	PartyID string
	// The amount to be deposited
	Amount *num.Uint
}

func NewBuiltinAssetDepositFromProto(p *vegapb.BuiltinAssetDeposit) (*BuiltinAssetDeposit, error) {
	amount := num.UintZero()
	if len(p.Amount) > 0 {
		var overflowed bool
		amount, overflowed = num.UintFromString(p.Amount, 10)
		if overflowed {
			return nil, errors.New("invalid amount")
		}
	}
	return &BuiltinAssetDeposit{
		VegaAssetID: p.VegaAssetId,
		PartyID:     p.PartyId,
		Amount:      amount,
	}, nil
}

func (b BuiltinAssetDeposit) IntoProto() *vegapb.BuiltinAssetDeposit {
	return &vegapb.BuiltinAssetDeposit{
		VegaAssetId: b.VegaAssetID,
		PartyId:     b.PartyID,
		Amount:      num.UintToString(b.Amount),
	}
}

func (b BuiltinAssetDeposit) String() string {
	return fmt.Sprintf(
		"party(%s) vegaAssetID(%s) amount(%s)",
		b.PartyID,
		b.VegaAssetID,
		stringer.PtrToString(b.Amount),
	)
}

func (b BuiltinAssetDeposit) GetVegaAssetID() string {
	return b.VegaAssetID
}

type BuiltinAssetWithdrawal struct {
	// A Vega network internal asset identifier
	VegaAssetID string
	// A Vega network party identifier (pub-key)
	PartyID string
	// The amount to be withdrawn
	Amount *num.Uint
}

func NewBuiltinAssetWithdrawalFromProto(p *vegapb.BuiltinAssetWithdrawal) (*BuiltinAssetWithdrawal, error) {
	amount := num.UintZero()
	if len(p.Amount) > 0 {
		var overflowed bool
		amount, overflowed = num.UintFromString(p.Amount, 10)
		if overflowed {
			return nil, errors.New("invalid amount")
		}
	}
	return &BuiltinAssetWithdrawal{
		VegaAssetID: p.VegaAssetId,
		PartyID:     p.PartyId,
		Amount:      amount,
	}, nil
}

func (b BuiltinAssetWithdrawal) IntoProto() *vegapb.BuiltinAssetWithdrawal {
	return &vegapb.BuiltinAssetWithdrawal{
		VegaAssetId: b.VegaAssetID,
		PartyId:     b.PartyID,
		Amount:      num.UintToString(b.Amount),
	}
}

func (b BuiltinAssetWithdrawal) String() string {
	return fmt.Sprintf(
		"partyID(%s) vegaAssetID(%s) amount(%s)",
		b.PartyID,
		b.VegaAssetID,
		stringer.PtrToString(b.Amount),
	)
}

func (b BuiltinAssetWithdrawal) GetVegaAssetID() string {
	return b.VegaAssetID
}

type ChainEventBuiltin struct {
	Builtin *BuiltinAssetEvent
}

func NewChainEventBuiltinFromProto(p *commandspb.ChainEvent_Builtin) (*ChainEventBuiltin, error) {
	c := ChainEventBuiltin{}
	var err error
	c.Builtin, err = NewBuiltinAssetEventFromProto(p.Builtin)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (c ChainEventBuiltin) IntoProto() *commandspb.ChainEvent_Builtin {
	ceb := &commandspb.ChainEvent_Builtin{
		Builtin: c.Builtin.IntoProto(),
	}
	return ceb
}

func (c ChainEventBuiltin) String() string {
	return fmt.Sprintf(
		"builtin(%s)",
		stringer.PtrToString(c.Builtin),
	)
}

type BuiltinAssetEvent struct {
	// Types that are valid to be assigned to Action:
	//	*BuiltinAssetEvent_Deposit
	//	*BuiltinAssetEvent_Withdrawal
	Action builtinAssetEventAction
}

type builtinAssetEventAction interface {
	isBuiltinAssetEvent()
	oneOfProto() interface{}
	String() string
}

func NewBuiltinAssetEventFromProto(p *vegapb.BuiltinAssetEvent) (*BuiltinAssetEvent, error) {
	var (
		ae  = &BuiltinAssetEvent{}
		err error
	)
	switch e := p.Action.(type) {
	case *vegapb.BuiltinAssetEvent_Deposit:
		ae.Action, err = NewBuiltinAssetEventDeposit(e)
		return ae, err
	case *vegapb.BuiltinAssetEvent_Withdrawal:
		ae.Action, err = NewBuiltinAssetEventWithdrawal(e)
		return ae, err
	default:
		return nil, errors.New("unknown asset event type")
	}
}

func (c BuiltinAssetEvent) IntoProto() *vegapb.BuiltinAssetEvent {
	action := c.Action.oneOfProto()
	ceb := &vegapb.BuiltinAssetEvent{}
	switch a := action.(type) {
	case *vegapb.BuiltinAssetEvent_Deposit:
		ceb.Action = a
	case *vegapb.BuiltinAssetEvent_Withdrawal:
		ceb.Action = a
	}
	return ceb
}

func (c BuiltinAssetEvent) String() string {
	return fmt.Sprintf(
		"action(%s)",
		stringer.ObjToString(c.Action),
	)
}

type BuiltinAssetEventDeposit struct {
	Deposit *BuiltinAssetDeposit
}

func (b BuiltinAssetEventDeposit) String() string {
	return fmt.Sprintf(
		"deposit(%s)",
		stringer.PtrToString(b.Deposit),
	)
}

func NewBuiltinAssetEventDeposit(p *vegapb.BuiltinAssetEvent_Deposit) (*BuiltinAssetEventDeposit, error) {
	dep, err := NewBuiltinAssetDepositFromProto(p.Deposit)
	if err != nil {
		return nil, err
	}
	return &BuiltinAssetEventDeposit{
		Deposit: dep,
	}, nil
}

func (b BuiltinAssetEventDeposit) IntoProto() *vegapb.BuiltinAssetEvent_Deposit {
	p := &vegapb.BuiltinAssetEvent_Deposit{
		Deposit: b.Deposit.IntoProto(),
	}
	return p
}

func (b BuiltinAssetEventDeposit) isBuiltinAssetEvent() {}

func (b BuiltinAssetEventDeposit) oneOfProto() interface{} {
	return b.IntoProto()
}

type BuiltinAssetEventWithdrawal struct {
	Withdrawal *BuiltinAssetWithdrawal
}

func (b BuiltinAssetEventWithdrawal) String() string {
	return fmt.Sprintf(
		"withdrawal(%s)",
		stringer.PtrToString(b.Withdrawal),
	)
}

func NewBuiltinAssetEventWithdrawal(p *vegapb.BuiltinAssetEvent_Withdrawal) (*BuiltinAssetEventWithdrawal, error) {
	withdrawal, err := NewBuiltinAssetWithdrawalFromProto(p.Withdrawal)
	if err != nil {
		return nil, err
	}
	return &BuiltinAssetEventWithdrawal{
		Withdrawal: withdrawal,
	}, nil
}

func (b BuiltinAssetEventWithdrawal) IntoProto() *vegapb.BuiltinAssetEvent_Withdrawal {
	p := &vegapb.BuiltinAssetEvent_Withdrawal{
		Withdrawal: b.Withdrawal.IntoProto(),
	}
	return p
}

func (b BuiltinAssetEventWithdrawal) isBuiltinAssetEvent() {}

func (b BuiltinAssetEventWithdrawal) oneOfProto() interface{} {
	return b.IntoProto()
}

type ERC20Event struct {
	// Index of the transaction
	Index uint64
	// The block in which the transaction was added
	Block   uint64
	ChainID string

	// The action
	//
	// Types that are valid to be assigned to Action:
	//	*ERC20EventAssetList
	//	*ERC20EventAssetDelist
	//	*ERC20EventDeposit
	//	*ERC20EventWithdrawal
	//	*ERC20EventAssetLimitsUpdated
	//	*ERC20BridgeStopped
	//	*ERC20BridgeRemoved
	Action erc20EventAction
}

type erc20EventAction interface {
	isErc20EventAction()
	oneOfProto() interface{}
	String() string
}

func NewERC20Event(p *vegapb.ERC20Event) (*ERC20Event, error) {
	e := ERC20Event{
		Index:   p.Index,
		Block:   p.Block,
		ChainID: p.ChainId,
	}

	var err error
	switch a := p.Action.(type) {
	case *vegapb.ERC20Event_AssetDelist:
		e.Action = NewERC20EventAssetDelist(a)
		return &e, nil
	case *vegapb.ERC20Event_AssetList:
		e.Action = NewERC20EventAssetList(a)
		return &e, nil
	case *vegapb.ERC20Event_Deposit:
		e.Action, err = NewERC20EventDeposit(a)
		if err != nil {
			return nil, err
		}
		return &e, nil
	case *vegapb.ERC20Event_Withdrawal:
		e.Action = NewERC20EventWithdrawal(a)
		return &e, nil
	case *vegapb.ERC20Event_AssetLimitsUpdated:
		e.Action = NewERC20EventAssetLimitsUpdated(a)
		return &e, nil
	case *vegapb.ERC20Event_BridgeStopped:
		e.Action = NewERC20EventBridgeStopped(a)
		return &e, nil
	case *vegapb.ERC20Event_BridgeResumed:
		e.Action = NewERC20EventBridgeResumed(a)
		return &e, nil
	default:
		return nil, errors.New("unknown erc20 event type")
	}
}

func (e ERC20Event) IntoProto() *vegapb.ERC20Event {
	p := &vegapb.ERC20Event{
		Index: e.Index,
		Block: e.Block,
	}

	switch a := e.Action.(type) {
	case *ERC20EventAssetDelist:
		p.Action = a.IntoProto()
	case *ERC20EventAssetList:
		p.Action = a.IntoProto()
	case *ERC20EventDeposit:
		p.Action = a.IntoProto()
	case *ERC20EventWithdrawal:
		p.Action = a.IntoProto()
	default:
		return nil
	}

	return p
}

func (e ERC20Event) String() string {
	return fmt.Sprintf(
		"index(%v) block(%v) action(%s)",
		e.Index,
		e.Block,
		stringer.ObjToString(e.Action),
	)
}

type ERC20EventAssetDelist struct {
	AssetDelist *ERC20AssetDelist
}

func (e ERC20EventAssetDelist) String() string {
	return fmt.Sprintf(
		"assetDelist(%s)",
		stringer.PtrToString(e.AssetDelist),
	)
}

func (ERC20EventAssetDelist) isErc20EventAction() {}

func (e ERC20EventAssetDelist) oneOfProto() interface{} {
	return e.AssetDelist.IntoProto()
}

func NewERC20EventAssetDelist(p *vegapb.ERC20Event_AssetDelist) *ERC20EventAssetDelist {
	return &ERC20EventAssetDelist{
		AssetDelist: NewERC20AssetDelistFromProto(p.AssetDelist),
	}
}

func (e ERC20EventAssetDelist) IntoProto() *vegapb.ERC20Event_AssetDelist {
	return &vegapb.ERC20Event_AssetDelist{
		AssetDelist: e.AssetDelist.IntoProto(),
	}
}

type ERC20AssetDelist struct {
	// The Vega network internal identifier of the asset
	VegaAssetID string
}

func NewERC20AssetDelistFromProto(p *vegapb.ERC20AssetDelist) *ERC20AssetDelist {
	return &ERC20AssetDelist{
		VegaAssetID: p.VegaAssetId,
	}
}

func (e ERC20AssetDelist) IntoProto() *vegapb.ERC20AssetDelist {
	return &vegapb.ERC20AssetDelist{
		VegaAssetId: e.VegaAssetID,
	}
}

func (e ERC20AssetDelist) String() string {
	return fmt.Sprintf("vegaAssetID(%s)", e.VegaAssetID)
}

type ERC20EventAssetList struct {
	AssetList *ERC20AssetList
}

func (ERC20EventAssetList) isErc20EventAction() {}

func (e ERC20EventAssetList) oneOfProto() interface{} {
	return e.AssetList.IntoProto()
}

func (e ERC20EventAssetList) String() string {
	return fmt.Sprintf(
		"assetList(%s)",
		stringer.PtrToString(e.AssetList),
	)
}

func NewERC20EventAssetList(p *vegapb.ERC20Event_AssetList) *ERC20EventAssetList {
	return &ERC20EventAssetList{
		AssetList: NewERC20AssetListFromProto(p.AssetList),
	}
}

func (e ERC20EventAssetList) IntoProto() *vegapb.ERC20Event_AssetList {
	return &vegapb.ERC20Event_AssetList{
		AssetList: e.AssetList.IntoProto(),
	}
}

type ERC20AssetList struct {
	// The Vega network internal identifier of the asset
	VegaAssetID string
	// ethereum address of the asset
	AssetSource string
}

func NewERC20AssetListFromProto(p *vegapb.ERC20AssetList) *ERC20AssetList {
	return &ERC20AssetList{
		VegaAssetID: p.VegaAssetId,
		AssetSource: p.AssetSource,
	}
}

func (e ERC20AssetList) IntoProto() *vegapb.ERC20AssetList {
	return &vegapb.ERC20AssetList{
		VegaAssetId: e.VegaAssetID,
	}
}

func (e ERC20AssetList) String() string {
	return fmt.Sprintf(
		"vegaAssetID(%s)",
		e.VegaAssetID,
	)
}

func (e ERC20AssetList) GetVegaAssetID() string {
	return e.VegaAssetID
}

type ERC20EventWithdrawal struct {
	Withdrawal *ERC20Withdrawal
}

func (ERC20EventWithdrawal) isErc20EventAction() {}

func (e ERC20EventWithdrawal) oneOfProto() interface{} {
	return e.Withdrawal.IntoProto()
}

func (e ERC20EventWithdrawal) String() string {
	return fmt.Sprintf(
		"withdrawal(%s)",
		stringer.PtrToString(e.Withdrawal),
	)
}

func NewERC20EventWithdrawal(p *vegapb.ERC20Event_Withdrawal) *ERC20EventWithdrawal {
	return &ERC20EventWithdrawal{
		Withdrawal: NewERC20WithdrawalFromProto(p.Withdrawal),
	}
}

func (e ERC20EventWithdrawal) IntoProto() *vegapb.ERC20Event_Withdrawal {
	return &vegapb.ERC20Event_Withdrawal{
		Withdrawal: e.Withdrawal.IntoProto(),
	}
}

type ERC20Withdrawal struct {
	// The Vega network internal identifier of the asset
	VegaAssetID string
	// The target Ethereum wallet address
	TargetEthereumAddress string
	// The reference nonce used for the transaction
	ReferenceNonce string
}

func NewERC20WithdrawalFromProto(p *vegapb.ERC20Withdrawal) *ERC20Withdrawal {
	return &ERC20Withdrawal{
		VegaAssetID:           p.VegaAssetId,
		TargetEthereumAddress: p.TargetEthereumAddress,
		ReferenceNonce:        p.ReferenceNonce,
	}
}

func (e ERC20Withdrawal) IntoProto() *vegapb.ERC20Withdrawal {
	return &vegapb.ERC20Withdrawal{
		VegaAssetId:           e.VegaAssetID,
		TargetEthereumAddress: e.TargetEthereumAddress,
		ReferenceNonce:        e.ReferenceNonce,
	}
}

func (e ERC20Withdrawal) String() string {
	return fmt.Sprintf(
		"vegaAssetID(%s) referenceNonce(%s) targetEthereumAddress(%s)",
		e.VegaAssetID,
		e.ReferenceNonce,
		e.TargetEthereumAddress,
	)
}

func (e ERC20Withdrawal) GetVegaAssetID() string {
	return e.VegaAssetID
}

type ERC20EventDeposit struct {
	Deposit *ERC20Deposit
}

func (e ERC20EventDeposit) String() string {
	return fmt.Sprintf(
		"deposit(%s)",
		stringer.PtrToString(e.Deposit),
	)
}

func (ERC20EventDeposit) isErc20EventAction() {}

func (e ERC20EventDeposit) oneOfProto() interface{} {
	return e.Deposit.IntoProto()
}

func NewERC20EventDeposit(p *vegapb.ERC20Event_Deposit) (*ERC20EventDeposit, error) {
	e := ERC20EventDeposit{}
	var err error
	e.Deposit, err = NewERC20DepositFromProto(p.Deposit)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (e ERC20EventDeposit) IntoProto() *vegapb.ERC20Event_Deposit {
	p := vegapb.ERC20Event_Deposit{
		Deposit: e.Deposit.IntoProto(),
	}
	return &p
}

type ERC20Deposit struct {
	// The vega network internal identifier of the asset
	VegaAssetID string
	// The Ethereum wallet that initiated the deposit
	SourceEthereumAddress string
	// The Vega party identifier (pub-key) which is the target of the deposit
	TargetPartyID string
	// The amount to be deposited
	Amount *num.Uint
}

func NewERC20DepositFromProto(p *vegapb.ERC20Deposit) (*ERC20Deposit, error) {
	e := ERC20Deposit{
		VegaAssetID:           p.VegaAssetId,
		SourceEthereumAddress: p.SourceEthereumAddress,
		TargetPartyID:         p.TargetPartyId,
	}
	if len(p.Amount) > 0 {
		var failed bool
		e.Amount, failed = num.UintFromString(p.Amount, 10)
		if failed {
			return nil, fmt.Errorf("failed to convert numerical string to Uint: %v", p.Amount)
		}
	}
	return &e, nil
}

func (e ERC20Deposit) IntoProto() *vegapb.ERC20Deposit {
	return &vegapb.ERC20Deposit{
		VegaAssetId:           e.VegaAssetID,
		SourceEthereumAddress: e.SourceEthereumAddress,
		TargetPartyId:         e.TargetPartyID,
		Amount:                num.UintToString(e.Amount),
	}
}

func (e ERC20Deposit) String() string {
	return fmt.Sprintf(
		"vegaAssetID(%s) targetPartyID(%s) amount(%s) sourceEthereumAddress(%s)",
		e.VegaAssetID,
		e.TargetPartyID,
		stringer.PtrToString(e.Amount),
		e.SourceEthereumAddress,
	)
}

func (e ERC20Deposit) GetVegaAssetID() string {
	return e.VegaAssetID
}

type ERC20EventAssetLimitsUpdated struct {
	AssetLimitsUpdated *ERC20AssetLimitsUpdated
}

func (ERC20EventAssetLimitsUpdated) isErc20EventAction() {}

func (e ERC20EventAssetLimitsUpdated) oneOfProto() interface{} {
	return e.AssetLimitsUpdated.IntoProto()
}

func (e ERC20EventAssetLimitsUpdated) String() string {
	return fmt.Sprintf(
		"assetLimitsUpdated(%s)",
		stringer.PtrToString(e.AssetLimitsUpdated),
	)
}

func NewERC20EventAssetLimitsUpdated(p *vegapb.ERC20Event_AssetLimitsUpdated) *ERC20EventAssetLimitsUpdated {
	return &ERC20EventAssetLimitsUpdated{
		AssetLimitsUpdated: NewERC20AssetLimitsUpdatedFromProto(p.AssetLimitsUpdated),
	}
}

func (e ERC20EventAssetLimitsUpdated) IntoProto() *vegapb.ERC20Event_AssetLimitsUpdated {
	return &vegapb.ERC20Event_AssetLimitsUpdated{
		AssetLimitsUpdated: e.AssetLimitsUpdated.IntoProto(),
	}
}

type ERC20AssetLimitsUpdated struct {
	VegaAssetID           string
	SourceEthereumAddress string
	LifetimeLimits        *num.Uint
	WithdrawThreshold     *num.Uint
}

func NewERC20AssetLimitsUpdatedFromProto(p *vegapb.ERC20AssetLimitsUpdated) *ERC20AssetLimitsUpdated {
	lifetimeLimits, _ := num.UintFromString(p.LifetimeLimits, 10)
	withdrawThreshold, _ := num.UintFromString(p.WithdrawThreshold, 10)
	return &ERC20AssetLimitsUpdated{
		VegaAssetID:           p.VegaAssetId,
		SourceEthereumAddress: p.SourceEthereumAddress,
		LifetimeLimits:        lifetimeLimits,
		WithdrawThreshold:     withdrawThreshold,
	}
}

func (e ERC20AssetLimitsUpdated) IntoProto() *vegapb.ERC20AssetLimitsUpdated {
	return &vegapb.ERC20AssetLimitsUpdated{
		VegaAssetId:           e.VegaAssetID,
		SourceEthereumAddress: e.SourceEthereumAddress,
		LifetimeLimits:        num.UintToString(e.LifetimeLimits),
		WithdrawThreshold:     num.UintToString(e.WithdrawThreshold),
	}
}

func (e ERC20AssetLimitsUpdated) String() string {
	return fmt.Sprintf(
		"vegaAssetID(%s) sourceEthereumAddress(%s) lifetimeLimits(%s) withdrawThreshold(%s)",
		e.VegaAssetID,
		e.SourceEthereumAddress,
		stringer.PtrToString(e.LifetimeLimits),
		stringer.PtrToString(e.WithdrawThreshold),
	)
}

func (e ERC20AssetLimitsUpdated) GetVegaAssetID() string {
	return e.VegaAssetID
}

type ERC20EventBridgeStopped struct {
	BridgeStopped bool
}

func (ERC20EventBridgeStopped) isErc20EventAction() {}

func (e ERC20EventBridgeStopped) oneOfProto() interface{} {
	return e.IntoProto()
}

func (e ERC20EventBridgeStopped) String() string {
	return fmt.Sprintf(
		"bridgeStopped(%v)",
		e.BridgeStopped,
	)
}

func NewERC20EventBridgeStopped(p *vegapb.ERC20Event_BridgeStopped) *ERC20EventBridgeStopped {
	return &ERC20EventBridgeStopped{
		BridgeStopped: p.BridgeStopped,
	}
}

func (e ERC20EventBridgeStopped) IntoProto() *vegapb.ERC20Event_BridgeStopped {
	return &vegapb.ERC20Event_BridgeStopped{
		BridgeStopped: e.BridgeStopped,
	}
}

type ERC20EventBridgeResumed struct {
	BridgeResumed bool
}

func (ERC20EventBridgeResumed) isErc20EventAction() {}

func (e ERC20EventBridgeResumed) oneOfProto() interface{} {
	return e.IntoProto()
}

func (e ERC20EventBridgeResumed) String() string {
	return fmt.Sprintf(
		"bridgeResumed(%v)",
		e.BridgeResumed,
	)
}

func NewERC20EventBridgeResumed(p *vegapb.ERC20Event_BridgeResumed) *ERC20EventBridgeResumed {
	return &ERC20EventBridgeResumed{
		BridgeResumed: p.BridgeResumed,
	}
}

func (e ERC20EventBridgeResumed) IntoProto() *vegapb.ERC20Event_BridgeResumed {
	return &vegapb.ERC20Event_BridgeResumed{
		BridgeResumed: e.BridgeResumed,
	}
}
