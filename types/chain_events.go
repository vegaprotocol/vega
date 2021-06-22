//lint:file-ignore ST1003 Ignore underscores in names, this is straight copied from the proto package to ease introducing the domain types

package types

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type Deposit = proto.Deposit
type Withdrawal = proto.Withdrawal
type WithdrawExt = proto.WithdrawExt
type WithdrawExt_Erc20 = proto.WithdrawExt_Erc20

type Erc20WithdrawExt = proto.Erc20WithdrawExt
type ChainEvent_Btc = commandspb.ChainEvent_Btc
type ChainEvent_Validator = commandspb.ChainEvent_Validator
type BuiltinAssetEvent_Deposit = proto.BuiltinAssetEvent_Deposit
type BuiltinAssetEvent_Withdrawal = proto.BuiltinAssetEvent_Withdrawal

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
	return c.IntoProto().Erc20.String()
}

type BuiltinAssetDeposit struct {
	// A Vega network internal asset identifier
	VegaAssetId string
	// A Vega party identifier (pub-key)
	PartyId string
	// The amount to be deposited
	Amount *num.Uint
}

func NewBuiltinAssetDepositFromProto(p *proto.BuiltinAssetDeposit) (*BuiltinAssetDeposit, error) {
	b := BuiltinAssetDeposit{
		VegaAssetId: p.VegaAssetId,
		PartyId:     p.PartyId,
		Amount:      num.NewUint(p.Amount),
	}
	return &b, nil
}

func (b BuiltinAssetDeposit) IntoProto() *proto.BuiltinAssetDeposit {
	bd := &proto.BuiltinAssetDeposit{
		VegaAssetId: b.VegaAssetId,
		PartyId:     b.PartyId,
		Amount:      b.Amount.Uint64(),
	}
	return bd
}

func (b BuiltinAssetDeposit) String() string {
	return b.IntoProto().String()
}

func (b BuiltinAssetDeposit) GetVegaAssetId() string {
	return b.VegaAssetId
}

type BuiltinAssetWithdrawal struct {
	// A Vega network internal asset identifier
	VegaAssetId string
	// A Vega network party identifier (pub-key)
	PartyId string
	// The amount to be withdrawn
	Amount *num.Uint
}

func NewBuiltinAssetWithdrawalFromProto(p *proto.BuiltinAssetWithdrawal) (*BuiltinAssetWithdrawal, error) {
	b := BuiltinAssetWithdrawal{
		VegaAssetId: p.VegaAssetId,
		PartyId:     p.PartyId,
		Amount:      num.NewUint(p.Amount),
	}
	return &b, nil
}

func (b BuiltinAssetWithdrawal) IntoProto() *proto.BuiltinAssetWithdrawal {
	bd := &proto.BuiltinAssetWithdrawal{
		VegaAssetId: b.VegaAssetId,
		PartyId:     b.PartyId,
		Amount:      b.Amount.Uint64(),
	}
	return bd
}

func (b BuiltinAssetWithdrawal) String() string {
	return b.IntoProto().String()
}

func (b BuiltinAssetWithdrawal) GetVegaAssetId() string {
	return b.VegaAssetId
}

type ChainEvent_Builtin struct {
	Builtin *BuiltinAssetEvent
}

func NewChainEventBuiltinFromProto(p *commandspb.ChainEvent_Builtin) (*ChainEvent_Builtin, error) {
	c := ChainEvent_Builtin{}
	var err error
	c.Builtin, err = NewBuiltinAssetEventFromProto(p.Builtin)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (c ChainEvent_Builtin) IntoProto() *commandspb.ChainEvent_Builtin {
	ceb := &commandspb.ChainEvent_Builtin{
		Builtin: c.Builtin.IntoProto(),
	}
	return ceb
}

func (c ChainEvent_Builtin) String() string {
	return c.IntoProto().Builtin.String()
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
}

func NewBuiltinAssetEventFromProto(p *proto.BuiltinAssetEvent) (*BuiltinAssetEvent, error) {
	ae := &BuiltinAssetEvent{}
	var err error
	switch e := p.Action.(type) {
	case *proto.BuiltinAssetEvent_Deposit:
		ae.Action, err = NewBuiltinAssetEventDeposit(e)
		if err != nil {
			return nil, err
		}
		return ae, nil
	case *proto.BuiltinAssetEvent_Withdrawal:
		ae.Action, err = NewBuiltinAssetEventWithdrawal(e)
		if err != nil {
			return nil, err
		}
		return ae, nil
	default:
		return nil, errors.New("Unknown asset event type")
	}
}

func (c BuiltinAssetEvent) IntoProto() *proto.BuiltinAssetEvent {
	action := c.Action.oneOfProto()
	ceb := &proto.BuiltinAssetEvent{}
	switch a := action.(type) {
	case *proto.BuiltinAssetEvent_Deposit:
		ceb.Action = a
	case *proto.BuiltinAssetEvent_Withdrawal:
		ceb.Action = a
	}
	return ceb
}

type BuiltinAssetEventDeposit struct {
	Deposit *BuiltinAssetDeposit
}

func NewBuiltinAssetEventDeposit(p *proto.BuiltinAssetEvent_Deposit) (*BuiltinAssetEventDeposit, error) {
	bd := BuiltinAssetEventDeposit{}
	var err error
	bd.Deposit, err = NewBuiltinAssetDepositFromProto(p.Deposit)
	if err != nil {
		return nil, err
	}
	return &bd, nil
}

func (b BuiltinAssetEventDeposit) IntoProto() *proto.BuiltinAssetEvent_Deposit {
	p := &proto.BuiltinAssetEvent_Deposit{
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

func NewBuiltinAssetEventWithdrawal(p *proto.BuiltinAssetEvent_Withdrawal) (*BuiltinAssetEventWithdrawal, error) {
	bd := BuiltinAssetEventWithdrawal{}
	var err error
	bd.Withdrawal, err = NewBuiltinAssetWithdrawalFromProto(p.Withdrawal)
	if err != nil {
		return nil, err
	}
	return &bd, nil
}

func (b BuiltinAssetEventWithdrawal) IntoProto() *proto.BuiltinAssetEvent_Withdrawal {
	p := &proto.BuiltinAssetEvent_Withdrawal{
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
	Block uint64
	// The action
	//
	// Types that are valid to be assigned to Action:
	//	*ERC20EventAssetList
	//	*ERC20EventAssetDelist
	//	*ERC20EventDeposit
	//	*ERC20EventWithdrawal
	Action erc20EventAction
}

type erc20EventAction interface {
	isErc20EventAction()
	oneOfProto() interface{}
}

func NewERC20Event(p *proto.ERC20Event) (*ERC20Event, error) {
	e := ERC20Event{
		Index: p.Index,
		Block: p.Block,
	}

	var err error
	switch a := p.Action.(type) {
	case *proto.ERC20Event_AssetDelist:
		e.Action, err = NewERC20EventAssetDelist(a)
		if err != nil {
			return nil, err
		}
		return &e, nil
	case *proto.ERC20Event_AssetList:
		e.Action, err = NewERC20EventAssetList(a)
		if err != nil {
			return nil, err
		}
		return &e, nil
	case *proto.ERC20Event_Deposit:
		e.Action, err = NewERC20EventDeposit(a)
		if err != nil {
			return nil, err
		}
		return &e, nil
	case *proto.ERC20Event_Withdrawal:
		e.Action, err = NewERC20EventWithdrawal(a)
		if err != nil {
			return nil, err
		}
		return &e, nil
	default:
		return nil, errors.New("Unknown erc20 event type")
	}
}

func (e ERC20Event) IntoProto() *proto.ERC20Event {
	p := &proto.ERC20Event{
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

type ERC20EventAssetDelist struct {
	AssetDelist *ERC20AssetDelist
}

func (ERC20EventAssetDelist) isErc20EventAction() {}
func (e ERC20EventAssetDelist) oneOfProto() interface{} {
	return e.AssetDelist.IntoProto()
}

func NewERC20EventAssetDelist(p *proto.ERC20Event_AssetDelist) (*ERC20EventAssetDelist, error) {
	e := ERC20EventAssetDelist{}
	var err error
	e.AssetDelist, err = NewERC20AssetDelistFromProto(p.AssetDelist)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (e ERC20EventAssetDelist) IntoProto() *proto.ERC20Event_AssetDelist {
	p := proto.ERC20Event_AssetDelist{
		AssetDelist: e.AssetDelist.IntoProto(),
	}
	return &p
}

type ERC20AssetDelist struct {
	// The Vega network internal identifier of the asset
	VegaAssetId string
}

func NewERC20AssetDelistFromProto(p *proto.ERC20AssetDelist) (*ERC20AssetDelist, error) {
	e := ERC20AssetDelist{
		VegaAssetId: p.VegaAssetId,
	}
	return &e, nil
}

func (e ERC20AssetDelist) IntoProto() *proto.ERC20AssetDelist {
	erc := &proto.ERC20AssetDelist{
		VegaAssetId: e.VegaAssetId,
	}
	return erc
}

func (e ERC20AssetDelist) String() string {
	return e.IntoProto().String()
}

type ERC20EventAssetList struct {
	AssetList *ERC20AssetList
}

func (ERC20EventAssetList) isErc20EventAction() {}
func (e ERC20EventAssetList) oneOfProto() interface{} {
	return e.AssetList.IntoProto()
}

func NewERC20EventAssetList(p *proto.ERC20Event_AssetList) (*ERC20EventAssetList, error) {
	e := ERC20EventAssetList{}
	var err error
	e.AssetList, err = NewERC20AssetListFromProto(p.AssetList)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (e ERC20EventAssetList) IntoProto() *proto.ERC20Event_AssetList {
	p := proto.ERC20Event_AssetList{
		AssetList: e.AssetList.IntoProto(),
	}
	return &p
}

type ERC20AssetList struct {
	// The Vega network internal identifier of the asset
	VegaAssetId string
}

func NewERC20AssetListFromProto(p *proto.ERC20AssetList) (*ERC20AssetList, error) {
	e := ERC20AssetList{
		VegaAssetId: p.VegaAssetId,
	}
	return &e, nil
}

func (e ERC20AssetList) IntoProto() *proto.ERC20AssetList {
	erc := &proto.ERC20AssetList{
		VegaAssetId: e.VegaAssetId,
	}
	return erc
}

func (e ERC20AssetList) String() string {
	return e.IntoProto().String()
}

func (e ERC20AssetList) GetVegaAssetId() string {
	return e.VegaAssetId
}

type ERC20EventWithdrawal struct {
	Withdrawal *ERC20Withdrawal
}

func (ERC20EventWithdrawal) isErc20EventAction() {}
func (e ERC20EventWithdrawal) oneOfProto() interface{} {
	return e.Withdrawal.IntoProto()
}

func NewERC20EventWithdrawal(p *proto.ERC20Event_Withdrawal) (*ERC20EventWithdrawal, error) {
	e := ERC20EventWithdrawal{}
	var err error
	e.Withdrawal, err = NewERC20WithdrawalFromProto(p.Withdrawal)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (e ERC20EventWithdrawal) IntoProto() *proto.ERC20Event_Withdrawal {
	p := proto.ERC20Event_Withdrawal{
		Withdrawal: e.Withdrawal.IntoProto(),
	}
	return &p
}

type ERC20Withdrawal struct {
	// The Vega network internal identifier of the asset
	VegaAssetId string
	// The target Ethereum wallet address
	TargetEthereumAddress string
	// The reference nonce used for the transaction
	ReferenceNonce string
}

func NewERC20WithdrawalFromProto(p *proto.ERC20Withdrawal) (*ERC20Withdrawal, error) {
	e := ERC20Withdrawal{
		VegaAssetId:           p.VegaAssetId,
		TargetEthereumAddress: p.TargetEthereumAddress,
		ReferenceNonce:        p.ReferenceNonce,
	}
	return &e, nil
}

func (e ERC20Withdrawal) IntoProto() *proto.ERC20Withdrawal {
	erc := &proto.ERC20Withdrawal{
		VegaAssetId:           e.VegaAssetId,
		TargetEthereumAddress: e.TargetEthereumAddress,
		ReferenceNonce:        e.ReferenceNonce,
	}
	return erc
}

func (e ERC20Withdrawal) String() string {
	return e.IntoProto().String()
}

func (e ERC20Withdrawal) GetVegaAssetId() string {
	return e.VegaAssetId
}

type ERC20EventDeposit struct {
	Deposit *ERC20Deposit
}

func (ERC20EventDeposit) isErc20EventAction() {}
func (e ERC20EventDeposit) oneOfProto() interface{} {
	return e.Deposit.IntoProto()
}

func NewERC20EventDeposit(p *proto.ERC20Event_Deposit) (*ERC20EventDeposit, error) {
	e := ERC20EventDeposit{}
	var err error
	e.Deposit, err = NewERC20DepositFromProto(p.Deposit)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (e ERC20EventDeposit) IntoProto() *proto.ERC20Event_Deposit {
	p := proto.ERC20Event_Deposit{
		Deposit: e.Deposit.IntoProto(),
	}
	return &p
}

type ERC20Deposit struct {
	// The vega network internal identifier of the asset
	VegaAssetId string
	// The Ethereum wallet that initiated the deposit
	SourceEthereumAddress string
	// The Vega party identifier (pub-key) which is the target of the deposit
	TargetPartyId string
	// The amount to be deposited
	Amount *num.Uint
}

func NewERC20DepositFromProto(p *proto.ERC20Deposit) (*ERC20Deposit, error) {
	e := ERC20Deposit{
		VegaAssetId:           p.VegaAssetId,
		SourceEthereumAddress: p.SourceEthereumAddress,
		TargetPartyId:         p.TargetPartyId,
	}
	var failed bool
	e.Amount, failed = num.UintFromString(p.Amount, 10)
	if failed {
		return nil, fmt.Errorf("Failed to convert numerical string to Uint: %v", p.Amount)
	}
	return &e, nil
}

func (e ERC20Deposit) IntoProto() *proto.ERC20Deposit {
	erc := &proto.ERC20Deposit{
		VegaAssetId:           e.VegaAssetId,
		SourceEthereumAddress: e.SourceEthereumAddress,
		TargetPartyId:         e.TargetPartyId,
		Amount:                e.Amount.String(),
	}
	return erc
}

func (e ERC20Deposit) String() string {
	return e.IntoProto().String()
}

func (e ERC20Deposit) GetVegaAssetId() string {
	return e.VegaAssetId
}
