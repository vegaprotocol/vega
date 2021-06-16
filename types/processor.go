//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/ptypes/wrappers"
)

//type WithdrawSubmission = commandspb.WithdrawSubmission
//type OracleDataSubmission = commandspb.OracleDataSubmission
//type NodeRegistration = commandspb.NodeRegistration
//type NodeVote = commandspb.NodeVote
//type Transaction = proto.Transaction
//type ChainEvent = commandspb.ChainEvent
//type SignedBundle = proto.SignedBundle
//type NetworkParameter = proto.NetworkParameter
//type Signature = proto.Signature
//type Transaction_PubKey = proto.Transaction_PubKey

type OrderSubmission struct {
	// Market identifier for the order, required field
	MarketId string
	// Price for the order, the price is an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places,
	// , required field for limit orders, however it is not required for market orders
	Price *num.Uint
	// Size for the order, for example, in a futures market the size equals the number of contracts, cannot be negative
	Size uint64
	// Side for the order, e.g. SIDE_BUY or SIDE_SELL, required field - See [`Side`](#vega.Side)
	Side proto.Side
	// Time in force indicates how long an order will remain active before it is executed or expires, required field
	// - See [`Order.TimeInForce`](#vega.Order.TimeInForce)
	TimeInForce proto.Order_TimeInForce
	// Timestamp for when the order will expire, in nanoseconds since the epoch,
	// required field only for [`Order.TimeInForce`](#vega.Order.TimeInForce)`.TIME_IN_FORCE_GTT`
	// - See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`
	ExpiresAt int64
	// Type for the order, required field - See [`Order.Type`](#vega.Order.Type)
	Type proto.Order_Type
	// Reference given for the order, this is typically used to retrieve an order submitted through consensus, currently
	// set internally by the node to return a unique reference identifier for the order submission
	Reference string
	// Used to specify the details for a pegged order
	// - See [`PeggedOrder`](#vega.PeggedOrder)
	PeggedOrder *PeggedOrder
}

func (o OrderSubmission) IntoProto() *commandspb.OrderSubmission {
	p := &commandspb.OrderSubmission{
		MarketId: o.MarketId,
		// Need to update protobuf to use string TODO UINT
		Price:       o.Price.Uint64(),
		Size:        o.Size,
		Side:        o.Side,
		TimeInForce: o.TimeInForce,
		ExpiresAt:   o.ExpiresAt,
		Type:        o.Type,
		Reference:   o.Reference,
		PeggedOrder: o.PeggedOrder.IntoProto(),
	}
	return p
}

func (o *OrderSubmission) FromProto(p *commandspb.OrderSubmission) {
	o.MarketId = p.MarketId
	// Need to update protobuf to use string TODO UINT
	o.Price = num.NewUint(p.Price)
	o.Size = p.Size
	o.Side = p.Side
	o.TimeInForce = p.TimeInForce
	o.ExpiresAt = p.ExpiresAt
	o.Type = p.Type
	o.Reference = p.Reference
	o.PeggedOrder.FromProto(p.PeggedOrder)
}

func (o OrderSubmission) String() string {
	return o.IntoProto().String()
}

func (o OrderSubmission) IntoOrder(party string) *Order {
	order := &Order{
		MarketId:    o.MarketId,
		PartyId:     party,
		Side:        o.Side,
		Price:       o.Price,
		Size:        o.Size,
		Remaining:   o.Size,
		TimeInForce: o.TimeInForce,
		Type:        o.Type,
		Status:      proto.Order_STATUS_ACTIVE,
		ExpiresAt:   o.ExpiresAt,
		Reference:   o.Reference,
		PeggedOrder: o.PeggedOrder,
	}
	return order
}

type OrderCancellation struct {
	// Unique identifier for the order (set by the system after consensus), required field
	OrderId string
	// Market identifier for the order, required field
	MarketId string
}

func (o *OrderCancellation) FromProto(p *commandspb.OrderCancellation) {
	o.MarketId = p.MarketId
	o.OrderId = p.OrderId
}

func (o OrderCancellation) IntoProto() *commandspb.OrderCancellation {
	oc := &commandspb.OrderCancellation{
		OrderId:  o.OrderId,
		MarketId: o.MarketId,
	}
	return oc
}

func (o OrderCancellation) String() string {
	return o.IntoProto().String()
}

type OrderAmendment struct {
	// Order identifier, this is required to find the order and will not be updated, required field
	OrderId string
	// Market identifier, this is required to find the order and will not be updated
	MarketId string
	// Amend the price for the order, if the Price value is set, otherwise price will remain unchanged - See [`Price`](#vega.Price)
	Price *num.Uint
	// Amend the size for the order by the delta specified:
	// - To reduce the size from the current value set a negative integer value
	// - To increase the size from the current value, set a positive integer value
	// - To leave the size unchanged set a value of zero
	SizeDelta int64
	// Amend the expiry time for the order, if the Timestamp value is set, otherwise expiry time will remain unchanged
	// - See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`
	ExpiresAt int64
	// Amend the time in force for the order, set to TIME_IN_FORCE_UNSPECIFIED to remain unchanged
	// - See [`TimeInForce`](#api.VegaTimeResponse).`timestamp`
	TimeInForce proto.Order_TimeInForce
	// Amend the pegged order offset for the order
	PeggedOffset int64
	// Amend the pegged order reference for the order
	// - See [`PeggedReference`](#vega.PeggedReference)
	PeggedReference proto.PeggedReference
}

func (o *OrderAmendment) FromProto(p *commandspb.OrderAmendment) {
	o.OrderId = p.OrderId
	o.MarketId = p.MarketId
	// Needs to update the protobuf definition TODO UINT
	if p.Price != nil {
		o.Price = num.NewUint(p.Price.Value)
	}
	o.SizeDelta = p.SizeDelta
	if p.ExpiresAt != nil {
		o.ExpiresAt = p.ExpiresAt.Value
	}
	o.TimeInForce = p.TimeInForce
	if p.PeggedOffset != nil {
		o.PeggedOffset = p.PeggedOffset.Value
	}
	o.PeggedReference = p.PeggedReference
}

func (o OrderAmendment) IntoProto() *commandspb.OrderAmendment {
	oa := &commandspb.OrderAmendment{
		OrderId:         o.OrderId,
		MarketId:        o.MarketId,
		SizeDelta:       o.SizeDelta,
		TimeInForce:     o.TimeInForce,
		PeggedReference: o.PeggedReference,
	}
	if !o.Price.IsZero() {
		oa.Price = &proto.Price{Value: o.Price.Uint64()}
	}
	if o.PeggedOffset != 0 {
		oa.PeggedOffset = &wrappers.Int64Value{Value: o.PeggedOffset}
	}
	if o.ExpiresAt != 0 {
		oa.ExpiresAt = &proto.Timestamp{Value: o.ExpiresAt}
	}
	return oa
}

func (o OrderAmendment) String() string {
	return o.IntoProto().String()
}

type WithdrawSubmission struct {
	// The amount to be withdrawn
	Amount uint64
	// The asset we want to withdraw
	Asset string
	// Foreign chain specifics
	Ext *proto.WithdrawExt
}

func (w *WithdrawSubmission) FromProto(p *commandspb.WithdrawSubmission) {
	w.Amount = p.Amount
	w.Asset = p.Asset
	w.Ext = p.Ext
}

func (w WithdrawSubmission) IntoProto() *commandspb.WithdrawSubmission {
	ws := &commandspb.WithdrawSubmission{
		Amount: w.Amount,
		Asset:  w.Asset,
		Ext:    w.Ext,
	}
	return ws
}

func (w WithdrawSubmission) String() string {
	return w.IntoProto().String()
}

type OracleDataSubmission struct {
	// The source from which the data is coming from
	Source OracleDataSubmission_OracleSource
	// The data provided by the third party provider
	Payload []byte
}

func (o *OracleDataSubmission) FromProto(p *commandspb.OracleDataSubmission) {
	o.Source = p.Source
	copy(o.Payload, p.Payload)
}

func (o OracleDataSubmission) IntoProto() *commandspb.OracleDataSubmission {
	ods := &commandspb.OracleDataSubmission{
		Source: o.Source,
	}
	copy(ods.Payload, o.Payload)
	return ods
}

func (o OracleDataSubmission) String() string {
	return o.IntoProto().String()
}

type NodeRegistration struct {
	// Public key, required field
	PubKey []byte
	// Public key for the blockchain, required field
	ChainPubKey []byte
}

func (n *NodeRegistration) FromProto(p *commandspb.NodeRegistration) {
	copy(n.PubKey, p.PubKey)
	copy(n.ChainPubKey, p.ChainPubKey)
}

func (n NodeRegistration) IntoProto() *commandspb.NodeRegistration {
	nr := &commandspb.NodeRegistration{}
	copy(nr.PubKey, n.PubKey)
	copy(nr.ChainPubKey, n.ChainPubKey)
	return nr
}

func (n NodeRegistration) String() string {
	return n.IntoProto().String()
}

type NodeVote struct {
	// Public key, required field
	PubKey []byte
	// Reference, required field
	Reference string
}

func (n *NodeVote) FromProto(p *commandspb.NodeVote) {
	copy(n.PubKey, p.PubKey)
	n.Reference = p.Reference
}

func (n NodeVote) IntoProto() *commandspb.NodeVote {
	nr := &commandspb.NodeVote{
		Reference: n.Reference,
	}
	copy(nr.PubKey, n.PubKey)
	return nr
}

func (n NodeVote) String() string {
	return n.IntoProto().String()
}

type Transaction struct {
	// One of the set of Vega commands (proto marshalled)
	InputData []byte
	// A random number used to provide uniqueness and prevent against replay attack
	Nonce uint64
	// The block height associated to the transaction, this should always be current block height
	// of the node at the time of sending the Tx and block height is used as a mechanism
	// for replay protection
	BlockHeight uint64
	// The sender of the transaction,
	// any of the following would be valid:
	//
	// Types that are valid to be assigned to From:
	//	*Transaction_Address
	//	*Transaction_PubKey
	From isTransaction_From
}

func (t *Transaction) FromProto(p *proto.Transaction) {
	t.Nonce = p.Nonce
	t.BlockHeight = p.BlockHeight
	copy(t.InputData, p.InputData)
}

func (t Transaction) IntoProto() *proto.Transaction {
	tr := &proto.Transaction{
		Nonce:       t.Nonce,
		BlockHeight: t.BlockHeight,
	}
	copy(tr.InputData, t.InputData)
	return tr
}

func (t Transaction) String() string {
	return t.IntoProto().String()
}

type ChainEvent struct {
	// The identifier of the transaction in which the events happened, usually a hash
	TxId string
	// Arbitrary one-time integer used to prevent replay attacks
	Nonce uint64
	// The event
	//
	// Types that are valid to be assigned to Event:
	//	*ChainEvent_Builtin
	//	*ChainEvent_Erc20
	//	*ChainEvent_Btc
	//	*ChainEvent_Validator
	Event isChainEvent_Event
}

func (c *ChainEvent) FromProto(p *commandspb.ChainEvent) {
	c.TxId = p.TxId
	c.Nonce = p.Nonce
}

func (c ChainEvent) IntoProto() *commandspb.ChainEvent {
	ce := &commandspb.ChainEvent{
		TxId:  c.TxId,
		Nonce: c.Nonce,
	}
	return ce
}

func (c ChainEvent) String() string {
	return c.IntoProto().String()
}

type SignedBundle struct {
	// Transaction payload (proto marshalled)
	Tx []byte
	// The signature authenticating the transaction
	Sig *Signature
}

func (s *SignedBundle) FromProto(p *proto.SignedBundle) {
	copy(s.Tx, p.Tx)
	s.Sig.FromProto(p.Sig)
}

func (s SignedBundle) IntoProto() *proto.SignedBundle {
	sb := &proto.SignedBundle{
		Sig: s.Sig.IntoProto(),
	}
	copy(sb.Tx, s.Tx)
	return sb
}

func (s SignedBundle) String() string {
	return s.IntoProto().String()
}

type NetworkParameter struct {
	// The unique key
	Key string
	// The value for the network parameter
	Value string
}

func (n *NetworkParameter) FromProto(p *proto.NetworkParameter) {
	n.Key = p.Key
	n.Value = p.Value
}

func (n NetworkParameter) IntoProto() *proto.NetworkParameter {
	np := &proto.NetworkParameter{
		Key:   n.Key,
		Value: n.Value,
	}
	return np
}

func (n NetworkParameter) String() string {
	return n.IntoProto().String()
}

type Signature struct {
	// The bytes of the signature
	Sig []byte
	// The algorithm used to create the signature
	Algo string
	// The version of the signature used to create the signature
	Version uint64
}

func (s *Signature) FromProto(p *proto.Signature) {
	copy(s.Sig, p.Sig)
	s.Algo = p.Algo
	s.Version = p.Version
}

func (s Signature) IntoProto() *proto.Signature {
	sig := &proto.Signature{
		Algo:    s.Algo,
		Version: s.Version,
	}
	copy(sig.Sig, s.Sig)
	return sig
}

func (s Signature) String() string {
	return s.IntoProto().String()
}

type Transaction_PubKey struct {
	PubKey []byte
}

func (t *Transaction_PubKey) FromProto(p *proto.Transaction_PubKey) {
	copy(t.PubKey, p.PubKey)
}

func (t Transaction_PubKey) IntoProto() *proto.Transaction_PubKey {
	tpk := &proto.Transaction_PubKey{}
	copy(tpk.PubKey, t.PubKey)
	return tpk
}

func (t Transaction_PubKey) String() string {
	return fmt.Sprint(t.PubKey)
}
