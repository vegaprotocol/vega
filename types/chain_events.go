//lint:file-ignore ST1003 Ignore underscores in names, this is straight copied from the proto package to ease introducing the domain types

package types

import (
	"errors"
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type WithdrawExt = proto.WithdrawExt
type WithdrawExt_Erc20 = proto.WithdrawExt_Erc20
type Erc20WithdrawExt = proto.Erc20WithdrawExt
type ChainEvent_Btc = commandspb.ChainEvent_Btc
type ChainEvent_Validator = commandspb.ChainEvent_Validator
type BuiltinAssetEvent_Deposit = proto.BuiltinAssetEvent_Deposit
type BuiltinAssetEvent_Withdrawal = proto.BuiltinAssetEvent_Withdrawal

type WithdrawalStatus = proto.Withdrawal_Status

const (
	// WithdrawalStatusUnspecified Default value, always invalid
	WithdrawalStatusUnspecified WithdrawalStatus = 0
	// WithdrawalStatusOpen The withdrawal is open and being processed by the network
	WithdrawalStatusOpen WithdrawalStatus = 1
	// WithdrawalStatusCancelled The withdrawal have been cancelled
	WithdrawalStatusCancelled WithdrawalStatus = 2
	// WithdrawalStatusFinalized The withdrawal went through and is fully finalised, the funds are removed from the
	// Vega network and are unlocked on the foreign chain bridge, for example, on the Ethereum network
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
	// ExpirationDate The time until when the withdrawal is valid
	ExpirationDate int64
	// Ext Foreign chain specifics
	Ext *WithdrawExt
}

func (w *Withdrawal) IntoProto() *proto.Withdrawal {
	return &proto.Withdrawal{
		Id:                 w.ID,
		PartyId:            w.PartyID,
		Amount:             num.UintToString(w.Amount),
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

func WithdrawalFromProto(w *proto.Withdrawal) *Withdrawal {
	amt, _ := num.UintFromString(w.Amount, 10)
	return &Withdrawal{
		ID:             w.Id,
		PartyID:        w.PartyId,
		Amount:         amt,
		Asset:          w.Asset,
		Status:         w.Status,
		Ref:            w.Ref,
		TxHash:         w.TxHash,
		ExpirationDate: w.Expiry,
		CreationDate:   w.CreatedTimestamp,
		WithdrawalDate: w.WithdrawnTimestamp,
		Ext:            w.Ext,
	}
}

type DepositStatus = proto.Deposit_Status

const (
	// DepositStatusUnspecified Default value, always invalid
	DepositStatusUnspecified DepositStatus = 0
	// DepositStatusOpen The deposit is being processed by the network
	DepositStatusOpen DepositStatus = 1
	// DepositStatusCancelled The deposit has been cancelled by the network
	DepositStatusCancelled DepositStatus = 2
	// DepositStatusFinalized The deposit has been finalised and accounts have been updated
	DepositStatusFinalized DepositStatus = 3
)

// Deposit represent a deposit on to the Vega network
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

func (d *Deposit) IntoProto() *proto.Deposit {
	return &proto.Deposit{
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

func DepositFromProto(d *proto.Deposit) *Deposit {
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
	return c.IntoProto().Erc20.String()
}

type BuiltinAssetDeposit struct {
	// A Vega network internal asset identifier
	VegaAssetID string
	// A Vega party identifier (pub-key)
	PartyID string
	// The amount to be deposited
	Amount *num.Uint
}

func NewBuiltinAssetDepositFromProto(p *proto.BuiltinAssetDeposit) (*BuiltinAssetDeposit, error) {
	var amount = num.Zero()
	if len(p.Amount) > 0 {
		var overflowed = false
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

func (b BuiltinAssetDeposit) IntoProto() *proto.BuiltinAssetDeposit {
	return &proto.BuiltinAssetDeposit{
		VegaAssetId: b.VegaAssetID,
		PartyId:     b.PartyID,
		Amount:      num.UintToString(b.Amount),
	}
}

func (b BuiltinAssetDeposit) String() string {
	return b.IntoProto().String()
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

func NewBuiltinAssetWithdrawalFromProto(p *proto.BuiltinAssetWithdrawal) (*BuiltinAssetWithdrawal, error) {
	var amount = num.Zero()
	if len(p.Amount) > 0 {
		var overflowed = false
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

func (b BuiltinAssetWithdrawal) IntoProto() *proto.BuiltinAssetWithdrawal {
	return &proto.BuiltinAssetWithdrawal{
		VegaAssetId: b.VegaAssetID,
		PartyId:     b.PartyID,
		Amount:      num.UintToString(b.Amount),
	}
}

func (b BuiltinAssetWithdrawal) String() string {
	return b.IntoProto().String()
}

func (b BuiltinAssetWithdrawal) GetVegaAssetID() string {
	return b.VegaAssetID
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
	var (
		ae  = &BuiltinAssetEvent{}
		err error
	)
	switch e := p.Action.(type) {
	case *proto.BuiltinAssetEvent_Deposit:
		ae.Action, err = NewBuiltinAssetEventDeposit(e)
		return ae, err
	case *proto.BuiltinAssetEvent_Withdrawal:
		ae.Action, err = NewBuiltinAssetEventWithdrawal(e)
		return ae, err
	default:
		return nil, errors.New("unknown asset event type")
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
	dep, err := NewBuiltinAssetDepositFromProto(p.Deposit)
	if err != nil {
		return nil, err
	}
	return &BuiltinAssetEventDeposit{
		Deposit: dep,
	}, nil
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
	withdrawal, err := NewBuiltinAssetWithdrawalFromProto(p.Withdrawal)
	if err != nil {
		return nil, err
	}
	return &BuiltinAssetEventWithdrawal{
		Withdrawal: withdrawal,
	}, nil
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
		e.Action = NewERC20EventAssetDelist(a)
		return &e, nil
	case *proto.ERC20Event_AssetList:
		e.Action = NewERC20EventAssetList(a)
		return &e, nil
	case *proto.ERC20Event_Deposit:
		e.Action, err = NewERC20EventDeposit(a)
		if err != nil {
			return nil, err
		}
		return &e, nil
	case *proto.ERC20Event_Withdrawal:
		e.Action = NewERC20EventWithdrawal(a)
		return &e, nil
	default:
		return nil, errors.New("unknown erc20 event type")
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

func NewERC20EventAssetDelist(p *proto.ERC20Event_AssetDelist) *ERC20EventAssetDelist {
	return &ERC20EventAssetDelist{
		AssetDelist: NewERC20AssetDelistFromProto(p.AssetDelist),
	}
}

func (e ERC20EventAssetDelist) IntoProto() *proto.ERC20Event_AssetDelist {
	return &proto.ERC20Event_AssetDelist{
		AssetDelist: e.AssetDelist.IntoProto(),
	}
}

type ERC20AssetDelist struct {
	// The Vega network internal identifier of the asset
	VegaAssetId string
}

func NewERC20AssetDelistFromProto(p *proto.ERC20AssetDelist) *ERC20AssetDelist {
	return &ERC20AssetDelist{
		VegaAssetId: p.VegaAssetId,
	}
}

func (e ERC20AssetDelist) IntoProto() *proto.ERC20AssetDelist {
	return &proto.ERC20AssetDelist{
		VegaAssetId: e.VegaAssetId,
	}
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

func NewERC20EventAssetList(p *proto.ERC20Event_AssetList) *ERC20EventAssetList {
	return &ERC20EventAssetList{
		AssetList: NewERC20AssetListFromProto(p.AssetList),
	}
}

func (e ERC20EventAssetList) IntoProto() *proto.ERC20Event_AssetList {
	return &proto.ERC20Event_AssetList{
		AssetList: e.AssetList.IntoProto(),
	}
}

type ERC20AssetList struct {
	// The Vega network internal identifier of the asset
	VegaAssetID string
}

func NewERC20AssetListFromProto(p *proto.ERC20AssetList) *ERC20AssetList {
	return &ERC20AssetList{
		VegaAssetID: p.VegaAssetId,
	}
}

func (e ERC20AssetList) IntoProto() *proto.ERC20AssetList {
	return &proto.ERC20AssetList{
		VegaAssetId: e.VegaAssetID,
	}
}

func (e ERC20AssetList) String() string {
	return e.IntoProto().String()
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

func NewERC20EventWithdrawal(p *proto.ERC20Event_Withdrawal) *ERC20EventWithdrawal {
	return &ERC20EventWithdrawal{
		Withdrawal: NewERC20WithdrawalFromProto(p.Withdrawal),
	}
}

func (e ERC20EventWithdrawal) IntoProto() *proto.ERC20Event_Withdrawal {
	return &proto.ERC20Event_Withdrawal{
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

func NewERC20WithdrawalFromProto(p *proto.ERC20Withdrawal) *ERC20Withdrawal {
	return &ERC20Withdrawal{
		VegaAssetID:           p.VegaAssetId,
		TargetEthereumAddress: p.TargetEthereumAddress,
		ReferenceNonce:        p.ReferenceNonce,
	}
}

func (e ERC20Withdrawal) IntoProto() *proto.ERC20Withdrawal {
	return &proto.ERC20Withdrawal{
		VegaAssetId:           e.VegaAssetID,
		TargetEthereumAddress: e.TargetEthereumAddress,
		ReferenceNonce:        e.ReferenceNonce,
	}
}

func (e ERC20Withdrawal) String() string {
	return e.IntoProto().String()
}

func (e ERC20Withdrawal) GetVegaAssetID() string {
	return e.VegaAssetID
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
	VegaAssetID string
	// The Ethereum wallet that initiated the deposit
	SourceEthereumAddress string
	// The Vega party identifier (pub-key) which is the target of the deposit
	TargetPartyID string
	// The amount to be deposited
	Amount *num.Uint
}

func NewERC20DepositFromProto(p *proto.ERC20Deposit) (*ERC20Deposit, error) {
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

func (e ERC20Deposit) IntoProto() *proto.ERC20Deposit {
	return &proto.ERC20Deposit{
		VegaAssetId:           e.VegaAssetID,
		SourceEthereumAddress: e.SourceEthereumAddress,
		TargetPartyId:         e.TargetPartyID,
		Amount:                num.UintToString(e.Amount),
	}
}

func (e ERC20Deposit) String() string {
	return e.IntoProto().String()
}

func (e ERC20Deposit) GetVegaAssetID() string {
	return e.VegaAssetID
}
