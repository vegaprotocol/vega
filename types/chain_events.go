//lint:file-ignore ST1003 Ignore underscores in names, this is straight copied from the proto package to ease introducing the domain types

package types

import (
	"errors"

	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type Deposit = proto.Deposit
type Withdrawal = proto.Withdrawal
type WithdrawExt = proto.WithdrawExt
type WithdrawExt_Erc20 = proto.WithdrawExt_Erc20

//type BuiltinAssetDeposit = proto.BuiltinAssetDeposit
type ERC20Deposit = proto.ERC20Deposit
type ERC20AssetList = proto.ERC20AssetList
type ERC20Withdrawal = proto.ERC20Withdrawal
type Erc20WithdrawExt = proto.Erc20WithdrawExt

//type ChainEvent_Builtin = commandspb.ChainEvent_Builtin
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

/*type Deposit struct {
	// Unique identifier for the deposit
	Id string
	// Status of the deposit
	Status Deposit_Status
	// Party identifier of the user initiating the deposit
	PartyId string
	// The Vega asset targeted by this deposit
	Asset string
	// The amount to be deposited
	Amount uint64
	// The hash of the transaction from the foreign chain
	TxHash string
	// Timestamp for when the Vega account was updated with the deposit
	CreditedTimestamp int64
	// Timestamp for when the deposit was created on the Vega network
	CreatedTimestamp int64
}

func (d *Deposit) FromProto(p *proto.Deposit) {
	d.Id = p.Id
	d.Status = p.Status
	d.PartyId = p.PartyId
	d.Asset = p.Asset
	d.Amount, _ = strconv.ParseUint(p.Amount, 10, 64)
	d.TxHash = p.TxHash
	d.CreditedTimestamp = p.CreditedTimestamp
	d.CreatedTimestamp = p.CreatedTimestamp
}

func (d Deposit) IntoProto() *proto.Deposit {
	dep := &proto.Deposit{
		Id:                d.Id,
		Status:            d.Status,
		PartyId:           d.PartyId,
		Asset:             d.Asset,
		Amount:            strconv.FormatUint(d.Amount, 10),
		TxHash:            d.TxHash,
		CreditedTimestamp: d.CreditedTimestamp,
		CreatedTimestamp:  d.CreatedTimestamp,
	}
	return dep
}

func (d Deposit) String() string {
	return d.IntoProto().String()
}

type Withdrawal struct {
	// Unique identifier for the withdrawal
	Id string
	// Unique party identifier of the user initiating the withdrawal
	PartyId string
	// The amount to be withdrawn
	Amount uint64
	// The asset we want to withdraw funds from
	Asset string
	// The status of the withdrawal
	Status Withdrawal_Status
	// The reference which is used by the foreign chain
	// to refer to this withdrawal
	Ref string
	// The time until when the withdrawal is valid
	Expiry int64
	// The hash of the foreign chain for this transaction
	TxHash string
	// Timestamp for when the network started to process this withdrawal
	CreatedTimestamp int64
	// Timestamp for when the withdrawal was finalised by the network
	WithdrawnTimestamp int64
	// Foreign chain specifics
	Ext *WithdrawExt
}

func (w *Withdrawal) FromProto(p *proto.Withdrawal) {
	w.Id = p.Id
	w.PartyId = p.PartyId
	w.Amount = p.Amount
	w.Asset = p.Asset
	w.Status = p.Status
	w.Ref = p.Ref
	w.Expiry = p.Expiry
	w.TxHash = p.TxHash
	w.CreatedTimestamp = p.CreatedTimestamp
	//	w.Ext.FromProto(p.Ext)
}

func (w Withdrawal) IntoProto() *proto.Withdrawal {
	dep := &proto.Withdrawal{
		Id:                 w.Id,
		PartyId:            w.PartyId,
		Amount:             w.Amount,
		Asset:              w.Asset,
		Status:             w.Status,
		Ref:                w.Ref,
		Expiry:             w.Expiry,
		TxHash:             w.TxHash,
		CreatedTimestamp:   w.CreatedTimestamp,
		WithdrawnTimestamp: w.WithdrawnTimestamp,
		//		Ext: nil,
	}
	return dep
}

func (w Withdrawal) String() string {
	return w.IntoProto().String()
}

type WithdrawExt struct {
	// Foreign chain specifics
	//
	// Types that are valid to be assigned to Ext:
	//	*WithdrawExt_Erc20
	Ext isWithdrawExt_Ext
}

func (w *WithdrawExt) FromProto(p *proto.WithdrawExt) {
}

func (w WithdrawExt) IntoProto() *proto.WithdrawExt {
	we := &proto.WithdrawExt{}
	return we
}

func (w WithdrawExt) String() string {
	return w.IntoProto().String()
}

type WithdrawExt_Erc20 struct {
	Erc20 *Erc20WithdrawExt
}

func (w *WithdrawExt_Erc20) FromProto(p *proto.WithdrawExt_Erc20) {
}

func (w WithdrawExt_Erc20) IntoProto() *proto.WithdrawExt_Erc20 {
	we := &proto.WithdrawExt_Erc20{}
	return we
}

func (w WithdrawExt_Erc20) String() string {
	return w.IntoProto().String()
}*/

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

/*type ERC20Deposit struct {
	// The vega network internal identifier of the asset
	VegaAssetId string
	// The Ethereum wallet that initiated the deposit
	SourceEthereumAddress string
	// The Vega party identifier (pub-key) which is the target of the deposit
	TargetPartyId string
	// The amount to be deposited
	Amount string
}

func (e *ERC20Deposit) FromProto(p *proto.ERC20Deposit) {
	e.VegaAssetId = p.VegaAssetId
	e.SourceEthereumAddress = p.SourceEthereumAddress
	e.TargetPartyId = p.TargetPartyId
	e.Amount = p.Amount
}

func (e ERC20Deposit) IntoProto() *proto.ERC20Deposit {
	erc := &proto.ERC20Deposit{
		VegaAssetId:           e.VegaAssetId,
		SourceEthereumAddress: e.SourceEthereumAddress,
		TargetPartyId:         e.TargetPartyId,
		Amount:                e.Amount,
	}
	return erc
}

func (e ERC20Deposit) String() string {
	return e.IntoProto().String()
}

type ERC20AssetList struct {
	// The Vega network internal identifier of the asset
	VegaAssetId string
}

func (e *ERC20AssetList) FromProto(p *proto.ERC20AssetList) {
	e.VegaAssetId = p.VegaAssetId
}

func (e ERC20AssetList) IntoProto() *proto.ERC20AssetList {
	erc := &proto.ERC20Deposit{
		VegaAssetId: e.VegaAssetId,
	}
	return erc
}

func (e ERC20AssetList) String() string {
	return e.IntoProto().String()
}

type ERC20Withdrawal struct {
	// The Vega network internal identifier of the asset
	VegaAssetId string
	// The target Ethereum wallet address
	TargetEthereumAddress string
	// The reference nonce used for the transaction
	ReferenceNonce string
}

func (e *ERC20Withdrawal) FromProto(p *proto.ERC20Withdrawal) {
	e.VegaAssetId = p.VegaAssetId
	e.TargetEthereumAddress = p.TargetEthereumAddress
	e.ReferenceNonce = p.ReferenceNonce
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

type Erc20WithdrawExt struct {
	// The address into which the bridge will release the funds
	ReceiverAddress string
}

func (e *Erc20WithdrawExt) FromProto(p *proto.Erc20WithdrawExt) {
	e.ReceiverAddress = p.ReceiverAddress
}

func (e Erc20WithdrawExt) IntoProto() *proto.Erc20WithdrawExt {
	erc := &proto.Erc20WithdrawExt{
		ReceiverAddress: e.ReceiverAddress,
	}
	return erc
}

func (e Erc20WithdrawExt) String() string {
	return e.IntoProto().String()
}*/

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
	Action builtinAssetEvent_Action
}

type builtinAssetEvent_Action interface {
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

/*type ChainEvent_Erc20 struct {
	Erc20 ERC20Event
}

func (c *ChainEvent_Erc20) FromProto(p *commandspb.ChainEvent_Erc20) {
	c.Erc20.FromProto(p.Erc20)
}

func (c ChainEvent_Erc20) IntoProto() *commandspb.ChainEvent_Erc20 {
	erc := &commandspb.ChainEvent_Erc20{
		Erc20: c.IntoProto(),
	}
	return erc
}

func (c ChainEvent_Erc20) String() string {
	return c.IntoProto().String()
}

type ChainEvent_Btc struct {
	Btc *BTCEvent
}

func (c *ChainEvent_Btc) FromProto(p *commandspb.ChainEvent_Btc) {
	c.Btc.FromProto(p.Btc)
}

func (c ChainEvent_Btc) IntoProto() *commandspb.ChainEvent_Btc {
	ce := &commandspb.ChainEvent_Btc{
		Btc: c.Btc.IntoProto(),
	}
	return ce
}

func (c ChainEvent_Btc) String() string {
	return c.IntoProto().String()
}

type ChainEvent_Validator struct {
	Validator *ValidatorEvent
}

func (c *ChainEvent_Validator) FromProto(p *commandspb.ChainEvent_Validator) {
	c.Validator.FromProto(p.Validator)
}

func (c ChainEvent_Validator) IntoProto() *commandspb.ChainEvent_Validator {
	ce := &commandspb.ChainEvent_Validator{
		Validator: c.Validator.IntoProto(),
	}
	return ce
}

func (c ChainEvent_Validator) String() string {
	return c.IntoProto().String()
}
*/
