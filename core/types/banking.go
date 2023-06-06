// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package types

import (
	"errors"
	"time"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type TransferStatus = eventspb.Transfer_Status

const (
	// Default value.
	TransferStatsUnspecified TransferStatus = eventspb.Transfer_STATUS_UNSPECIFIED
	// A pending transfer.
	TransferStatusPending TransferStatus = eventspb.Transfer_STATUS_PENDING
	// A finished transfer.
	TransferStatusDone TransferStatus = eventspb.Transfer_STATUS_DONE
	// A rejected transfer.
	TransferStatusRejected TransferStatus = eventspb.Transfer_STATUS_REJECTED
	// A stopped transfer.
	TransferStatusStopped TransferStatus = eventspb.Transfer_STATUS_STOPPED
	// A cancelled transfer.
	TransferStatusCancelled TransferStatus = eventspb.Transfer_STATUS_CANCELLED
)

var (
	ErrMissingTransferKind           = errors.New("missing transfer kind")
	ErrCannotTransferZeroFunds       = errors.New("cannot transfer zero funds")
	ErrInvalidFromAccount            = errors.New("invalid from account")
	ErrInvalidToAccount              = errors.New("invalid to account")
	ErrUnsupportedFromAccountType    = errors.New("unsupported from account type")
	ErrUnsupportedToAccountType      = errors.New("unsupported to account type")
	ErrEndEpochIsZero                = errors.New("end epoch is zero")
	ErrStartEpochIsZero              = errors.New("start epoch is zero")
	ErrInvalidFactor                 = errors.New("invalid factor")
	ErrStartEpochAfterEndEpoch       = errors.New("start epoch after end epoch")
	ErrInvalidToForRewardAccountType = errors.New("to party is invalid for reward account type")
)

type TransferCommandKind int

const (
	TransferCommandKindOneOff TransferCommandKind = iota
	TransferCommandKindRecurring
)

type TransferBase struct {
	ID              string
	From            string
	FromAccountType AccountType
	To              string
	ToAccountType   AccountType
	Asset           string
	Amount          *num.Uint
	Reference       string
	Status          TransferStatus
	Timestamp       time.Time
}

func (t *TransferBase) IsValid() error {
	if !vgcrypto.IsValidVegaPubKey(t.From) {
		return ErrInvalidFromAccount
	}
	if !vgcrypto.IsValidVegaPubKey(t.To) {
		return ErrInvalidToAccount
	}

	// ensure amount makes senses
	if t.Amount.IsZero() {
		return ErrCannotTransferZeroFunds
	}

	switch t.FromAccountType {
	case AccountTypeGeneral /*, AccountTypeLockedForStaking*/ :
		break
	default:
		return ErrUnsupportedFromAccountType
	}

	switch t.ToAccountType {
	case AccountTypeGlobalReward:
		if t.To != "0000000000000000000000000000000000000000000000000000000000000000" {
			return ErrInvalidToForRewardAccountType
		}
	case AccountTypeGeneral, AccountTypeLPFeeReward, AccountTypeMakerReceivedFeeReward, AccountTypeMakerPaidFeeReward, AccountTypeMarketProposerReward /*, AccountTypeLockedForStaking*/ :
		break
	default:
		return ErrUnsupportedToAccountType
	}

	return nil
}

type OneOffTransfer struct {
	*TransferBase
	DeliverOn *time.Time
}

func (o *OneOffTransfer) IsValid() error {
	if err := o.TransferBase.IsValid(); err != nil {
		return err
	}

	return nil
}

func OneOffTransferFromEvent(p *eventspb.Transfer) *OneOffTransfer {
	var deliverOn *time.Time
	if t := p.GetOneOff().GetDeliverOn(); t > 0 {
		d := time.Unix(0, t)
		deliverOn = &d
	}

	amount, overflow := num.UintFromString(p.Amount, 10)
	if overflow {
		// panic is alright here, this should come only from
		// a checkpoint, and it would mean the checkpoint is fucked
		// so executions is not possible.
		panic("invalid transfer amount")
	}

	return &OneOffTransfer{
		TransferBase: &TransferBase{
			ID:              p.Id,
			From:            p.From,
			FromAccountType: p.FromAccountType,
			To:              p.To,
			ToAccountType:   p.ToAccountType,
			Asset:           p.Asset,
			Amount:          amount,
			Reference:       p.Reference,
			Status:          p.Status,
			Timestamp:       time.Unix(0, p.Timestamp),
		},
		DeliverOn: deliverOn,
	}
}

func (o *OneOffTransfer) IntoEvent(reason *string) *eventspb.Transfer {
	out := &eventspb.Transfer{
		Id:              o.ID,
		From:            o.From,
		FromAccountType: o.FromAccountType,
		To:              o.To,
		ToAccountType:   o.ToAccountType,
		Asset:           o.Asset,
		Amount:          o.Amount.String(),
		Reference:       o.Reference,
		Status:          o.Status,
		Timestamp:       o.Timestamp.UnixNano(),
		Reason:          reason,
	}

	out.Kind = &eventspb.Transfer_OneOff{}
	if o.DeliverOn != nil {
		out.Kind = &eventspb.Transfer_OneOff{
			OneOff: &eventspb.OneOffTransfer{
				DeliverOn: o.DeliverOn.UnixNano(),
			},
		}
	}

	return out
}

type RecurringTransfer struct {
	*TransferBase
	StartEpoch       uint64
	EndEpoch         *uint64
	Factor           num.Decimal
	DispatchStrategy *vegapb.DispatchStrategy
}

func (r *RecurringTransfer) IsValid() error {
	if err := r.TransferBase.IsValid(); err != nil {
		return err
	}

	if r.EndEpoch != nil && *r.EndEpoch == 0 {
		return ErrEndEpochIsZero
	}
	if r.StartEpoch == 0 {
		return ErrStartEpochIsZero
	}

	if r.EndEpoch != nil && r.StartEpoch > *r.EndEpoch {
		return ErrStartEpochAfterEndEpoch
	}

	if r.Factor.Cmp(num.DecimalFromFloat(0)) <= 0 {
		return ErrInvalidFactor
	}

	return nil
}

func (r *RecurringTransfer) IntoEvent(reason *string) *eventspb.Transfer {
	var endEpoch *uint64
	if r.EndEpoch != nil {
		endEpoch = toPtr(*r.EndEpoch)
	}

	return &eventspb.Transfer{
		Id:              r.ID,
		From:            r.From,
		FromAccountType: r.FromAccountType,
		To:              r.To,
		ToAccountType:   r.ToAccountType,
		Asset:           r.Asset,
		Amount:          r.Amount.String(),
		Reference:       r.Reference,
		Status:          r.Status,
		Timestamp:       r.Timestamp.UnixNano(),
		Reason:          reason,
		Kind: &eventspb.Transfer_Recurring{
			Recurring: &eventspb.RecurringTransfer{
				StartEpoch:       r.StartEpoch,
				EndEpoch:         endEpoch,
				Factor:           r.Factor.String(),
				DispatchStrategy: r.DispatchStrategy,
			},
		},
	}
}

// Just a wrapper, use the Kind on a
// switch to access the proper value.
type TransferFunds struct {
	Kind      TransferCommandKind
	OneOff    *OneOffTransfer
	Recurring *RecurringTransfer
}

func NewTransferFromProto(id, from string, tf *commandspb.Transfer) (*TransferFunds, error) {
	base, err := newTransferBase(id, from, tf)
	if err != nil {
		return nil, err
	}
	switch tf.Kind.(type) {
	case *commandspb.Transfer_OneOff:
		return newOneOffTransfer(base, tf)
	case *commandspb.Transfer_Recurring:
		return newRecurringTransfer(base, tf)
	default:
		return nil, ErrMissingTransferKind
	}
}

func (t *TransferFunds) IntoEvent(reason *string) *eventspb.Transfer {
	switch t.Kind {
	case TransferCommandKindOneOff:
		return t.OneOff.IntoEvent(reason)
	case TransferCommandKindRecurring:
		return t.Recurring.IntoEvent(reason)
	default:
		panic("invalid transfer kind")
	}
}

func newTransferBase(id, from string, tf *commandspb.Transfer) (*TransferBase, error) {
	amount, overflowed := num.UintFromString(tf.Amount, 10)
	if overflowed {
		return nil, errors.New("invalid transfer amount")
	}

	return &TransferBase{
		ID:              id,
		From:            from,
		FromAccountType: tf.FromAccountType,
		To:              tf.To,
		ToAccountType:   tf.ToAccountType,
		Asset:           tf.Asset,
		Amount:          amount,
		Reference:       tf.Reference,
		Status:          TransferStatusPending,
	}, nil
}

func newOneOffTransfer(base *TransferBase, tf *commandspb.Transfer) (*TransferFunds, error) {
	var t *time.Time
	if tf.GetOneOff().GetDeliverOn() > 0 {
		tmpt := time.Unix(0, tf.GetOneOff().GetDeliverOn())
		t = &tmpt
	}

	return &TransferFunds{
		Kind: TransferCommandKindOneOff,
		OneOff: &OneOffTransfer{
			TransferBase: base,
			DeliverOn:    t,
		},
	}, nil
}

func newRecurringTransfer(base *TransferBase, tf *commandspb.Transfer) (*TransferFunds, error) {
	factor, err := num.DecimalFromString(tf.GetRecurring().GetFactor())
	if err != nil {
		return nil, err
	}
	var endEpoch *uint64
	if tf.GetRecurring().EndEpoch != nil {
		ee := tf.GetRecurring().GetEndEpoch()
		endEpoch = &ee
	}

	return &TransferFunds{
		Kind: TransferCommandKindRecurring,
		Recurring: &RecurringTransfer{
			TransferBase:     base,
			StartEpoch:       tf.GetRecurring().GetStartEpoch(),
			EndEpoch:         endEpoch,
			Factor:           factor,
			DispatchStrategy: tf.GetRecurring().DispatchStrategy,
		},
	}, nil
}

func RecurringTransferFromEvent(p *eventspb.Transfer) *RecurringTransfer {
	var endEpoch *uint64
	if p.GetRecurring().EndEpoch != nil {
		ee := p.GetRecurring().GetEndEpoch()
		endEpoch = &ee
	}

	factor, err := num.DecimalFromString(p.GetRecurring().GetFactor())
	if err != nil {
		panic("invalid decimal, should never happen")
	}

	amount, overflow := num.UintFromString(p.Amount, 10)
	if overflow {
		// panic is alright here, this should come only from
		// a checkpoint, and it would mean the checkpoint is fucked
		// so executions is not possible.
		panic("invalid transfer amount")
	}

	return &RecurringTransfer{
		TransferBase: &TransferBase{
			ID:              p.Id,
			From:            p.From,
			FromAccountType: p.FromAccountType,
			To:              p.To,
			ToAccountType:   p.ToAccountType,
			Asset:           p.Asset,
			Amount:          amount,
			Reference:       p.Reference,
			Status:          p.Status,
			Timestamp:       time.Unix(0, p.Timestamp),
		},
		StartEpoch:       p.GetRecurring().GetStartEpoch(),
		EndEpoch:         endEpoch,
		Factor:           factor,
		DispatchStrategy: p.GetRecurring().DispatchStrategy,
	}
}

type CancelTransferFunds struct {
	Party      string
	TransferID string
}

func NewCancelTransferFromProto(party string, p *commandspb.CancelTransfer) *CancelTransferFunds {
	return &CancelTransferFunds{
		Party:      party,
		TransferID: p.TransferId,
	}
}
