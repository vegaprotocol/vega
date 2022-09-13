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

type TransferInstructionStatus = eventspb.TransferInstruction_Status

const (
	// Default value.
	TransferInstructionStatsUnspecified TransferInstructionStatus = eventspb.TransferInstruction_STATUS_UNSPECIFIED
	// A pending transfer.
	TransferInstructionStatusPending TransferInstructionStatus = eventspb.TransferInstruction_STATUS_PENDING
	// A finished transfer.
	TransferInstructionStatusDone TransferInstructionStatus = eventspb.TransferInstruction_STATUS_DONE
	// A rejected transfer.
	TransferInstructionStatusRejected TransferInstructionStatus = eventspb.TransferInstruction_STATUS_REJECTED
	// A stopped transfer.
	TransferInstructionStatusStopped TransferInstructionStatus = eventspb.TransferInstruction_STATUS_STOPPED
	// A cancelled transfer.
	TransferInstructionStatusCancelled TransferInstructionStatus = eventspb.TransferInstruction_STATUS_CANCELLED
)

var (
	ErrMissingTransferInstructionKind     = errors.New("missing transfer instruction kind")
	ErrCannotTransferInstructionZeroFunds = errors.New("cannot transfer zero funds")
	ErrInvalidFromAccount                 = errors.New("invalid from account")
	ErrInvalidToAccount                   = errors.New("invalid to account")
	ErrUnsupportedFromAccountType         = errors.New("unsupported from account type")
	ErrUnsupportedToAccountType           = errors.New("unsupported to account type")
	ErrEndEpochIsZero                     = errors.New("end epoch is zero")
	ErrStartEpochIsZero                   = errors.New("start epoch is zero")
	ErrInvalidFactor                      = errors.New("invalid factor")
	ErrStartEpochAfterEndEpoch            = errors.New("start epoch after end epoch")
)

type TransferInstructionCommandKind int

const (
	TransferInstructionCommandKindOneOff TransferInstructionCommandKind = iota
	TransferInstructionCommandKindRecurring
)

type TransferInstructionBase struct {
	ID              string
	From            string
	FromAccountType AccountType
	To              string
	ToAccountType   AccountType
	Asset           string
	Amount          *num.Uint
	Reference       string
	Status          TransferInstructionStatus
	Timestamp       time.Time
}

func (t *TransferInstructionBase) IsValid() error {
	if !vgcrypto.IsValidVegaPubKey(t.From) {
		return ErrInvalidFromAccount
	}
	if !vgcrypto.IsValidVegaPubKey(t.To) {
		return ErrInvalidToAccount
	}

	// ensure amount makes senses
	if t.Amount.IsZero() {
		return ErrCannotTransferInstructionZeroFunds
	}

	switch t.FromAccountType {
	case AccountTypeGeneral /*, AccountTypeLockedForStaking*/ :
		break
	default:
		return ErrUnsupportedFromAccountType
	}

	switch t.ToAccountType {
	case AccountTypeGeneral, AccountTypeGlobalReward, AccountTypeLPFeeReward, AccountTypeMakerReceivedFeeReward, AccountTypeMakerPaidFeeReward, AccountTypeMarketProposerReward /*, AccountTypeLockedForStaking*/ :
		break
	default:
		return ErrUnsupportedToAccountType
	}

	return nil
}

type OneOffTransferInstruction struct {
	*TransferInstructionBase
	DeliverOn *time.Time
}

func (o *OneOffTransferInstruction) IsValid() error {
	if err := o.TransferInstructionBase.IsValid(); err != nil {
		return err
	}

	return nil
}

func OneOffTransferInstructionFromEvent(p *eventspb.TransferInstruction) *OneOffTransferInstruction {
	var deliverOn *time.Time
	if t := p.GetOneOff().GetDeliverOn(); t > 0 {
		d := time.Unix(t, 0)
		deliverOn = &d
	}

	amount, overflow := num.UintFromString(p.Amount, 10)
	if overflow {
		// panic is alright here, this should come only from
		// a checkpoint, and it would mean the checkpoint is fucked
		// so executions is not possible.
		panic("invalid transfer instruction amount")
	}

	return &OneOffTransferInstruction{
		TransferInstructionBase: &TransferInstructionBase{
			ID:              p.Id,
			From:            p.From,
			FromAccountType: p.FromAccountType,
			To:              p.To,
			ToAccountType:   p.ToAccountType,
			Asset:           p.Asset,
			Amount:          amount,
			Reference:       p.Reference,
			Status:          p.Status,
			Timestamp:       time.Unix(p.Timestamp/int64(time.Second), p.Timestamp%int64(time.Second)),
		},
		DeliverOn: deliverOn,
	}
}

func (o *OneOffTransferInstruction) IntoEvent() *eventspb.TransferInstruction {
	out := &eventspb.TransferInstruction{
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
	}

	if o.DeliverOn != nil {
		out.Kind = &eventspb.TransferInstruction_OneOff{
			OneOff: &eventspb.OneOffTransferInstruction{
				DeliverOn: o.DeliverOn.Unix(),
			},
		}
	}

	return out
}

type RecurringTransferInstruction struct {
	*TransferInstructionBase
	StartEpoch       uint64
	EndEpoch         *uint64
	Factor           num.Decimal
	DispatchStrategy *vegapb.DispatchStrategy
}

func (r *RecurringTransferInstruction) IsValid() error {
	if err := r.TransferInstructionBase.IsValid(); err != nil {
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

func (r *RecurringTransferInstruction) IntoEvent() *eventspb.TransferInstruction {
	var endEpoch *uint64
	if r.EndEpoch != nil {
		endEpoch = toPtr(*r.EndEpoch)
	}

	return &eventspb.TransferInstruction{
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
		Kind: &eventspb.TransferInstruction_Recurring{
			Recurring: &eventspb.RecurringTransferInstruction{
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
type TransferInstructionFunds struct {
	Kind      TransferInstructionCommandKind
	OneOff    *OneOffTransferInstruction
	Recurring *RecurringTransferInstruction
}

func NewTransferInstructionFromProto(id, from string, tf *commandspb.TransferInstruction) (*TransferInstructionFunds, error) {
	base, err := newTransferInstructionBase(id, from, tf)
	if err != nil {
		return nil, err
	}
	switch tf.Kind.(type) {
	case *commandspb.TransferInstruction_OneOff:
		return newOneOffTransferInstruction(base, tf)
	case *commandspb.TransferInstruction_Recurring:
		return newRecurringTransferInstruction(base, tf)
	default:
		return nil, ErrMissingTransferInstructionKind
	}
}

func (t *TransferInstructionFunds) IntoEvent() *eventspb.TransferInstruction {
	switch t.Kind {
	case TransferInstructionCommandKindOneOff:
		return t.OneOff.IntoEvent()
	case TransferInstructionCommandKindRecurring:
		return t.Recurring.IntoEvent()
	default:
		panic("invalid transfer kind")
	}
}

func newTransferInstructionBase(id, from string, tf *commandspb.TransferInstruction) (*TransferInstructionBase, error) {
	amount, overflowed := num.UintFromString(tf.Amount, 10)
	if overflowed {
		return nil, errors.New("invalid transfer amount")
	}

	return &TransferInstructionBase{
		ID:              id,
		From:            from,
		FromAccountType: tf.FromAccountType,
		To:              tf.To,
		ToAccountType:   tf.ToAccountType,
		Asset:           tf.Asset,
		Amount:          amount,
		Reference:       tf.Reference,
		Status:          TransferInstructionStatusPending,
	}, nil
}

func newOneOffTransferInstruction(base *TransferInstructionBase, tf *commandspb.TransferInstruction) (*TransferInstructionFunds, error) {
	var t *time.Time
	if tf.GetOneOff().GetDeliverOn() > 0 {
		tmpt := time.Unix(tf.GetOneOff().GetDeliverOn(), 0)
		t = &tmpt
	}

	return &TransferInstructionFunds{
		Kind: TransferInstructionCommandKindOneOff,
		OneOff: &OneOffTransferInstruction{
			TransferInstructionBase: base,
			DeliverOn:               t,
		},
	}, nil
}

func newRecurringTransferInstruction(base *TransferInstructionBase, tf *commandspb.TransferInstruction) (*TransferInstructionFunds, error) {
	factor, err := num.DecimalFromString(tf.GetRecurring().GetFactor())
	if err != nil {
		return nil, err
	}
	var endEpoch *uint64
	if tf.GetRecurring().EndEpoch != nil {
		ee := tf.GetRecurring().GetEndEpoch()
		endEpoch = &ee
	}

	return &TransferInstructionFunds{
		Kind: TransferInstructionCommandKindRecurring,
		Recurring: &RecurringTransferInstruction{
			TransferInstructionBase: base,
			StartEpoch:              tf.GetRecurring().GetStartEpoch(),
			EndEpoch:                endEpoch,
			Factor:                  factor,
			DispatchStrategy:        tf.GetRecurring().DispatchStrategy,
		},
	}, nil
}

func RecurringTransferInstructionFromEvent(p *eventspb.TransferInstruction) *RecurringTransferInstruction {
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

	return &RecurringTransferInstruction{
		TransferInstructionBase: &TransferInstructionBase{
			ID:              p.Id,
			From:            p.From,
			FromAccountType: p.FromAccountType,
			To:              p.To,
			ToAccountType:   p.ToAccountType,
			Asset:           p.Asset,
			Amount:          amount,
			Reference:       p.Reference,
			Status:          p.Status,
			Timestamp:       time.Unix(p.Timestamp/int64(time.Second), p.Timestamp%int64(time.Second)),
		},
		StartEpoch:       p.GetRecurring().GetStartEpoch(),
		EndEpoch:         endEpoch,
		Factor:           factor,
		DispatchStrategy: p.GetRecurring().DispatchStrategy,
	}
}

type CancelTransferInstructionFunds struct {
	Party      string
	TransferID string
}

func NewCancelTransferInstructionFromProto(party string, p *commandspb.CancelTransferInstruction) *CancelTransferInstructionFunds {
	return &CancelTransferInstructionFunds{
		Party:      party,
		TransferID: p.TransferId,
	}
}
