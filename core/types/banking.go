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
	"errors"
	"time"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	checkpointpb "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
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
	ErrInvalidFromDerivedKey         = errors.New("invalid from derived key")
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
	FromDerivedKey  *string
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

	// check for derived account transfer
	if t.FromDerivedKey != nil {
		if !vgcrypto.IsValidVegaPubKey(*t.FromDerivedKey) {
			return ErrInvalidFromDerivedKey
		}

		if t.FromAccountType != AccountTypeVestedRewards {
			return ErrUnsupportedFromAccountType
		}

		if t.ToAccountType != AccountTypeGeneral {
			return ErrUnsupportedToAccountType
		}
	}

	// check for any other transfers
	switch t.FromAccountType {
	case AccountTypeGeneral, AccountTypeVestedRewards /*, AccountTypeLockedForStaking*/ :
		break
	default:
		return ErrUnsupportedFromAccountType
	}

	switch t.ToAccountType {
	case AccountTypeGlobalReward, AccountTypeNetworkTreasury:
		if t.To != "0000000000000000000000000000000000000000000000000000000000000000" {
			return ErrInvalidToForRewardAccountType
		}
	case AccountTypeGeneral, AccountTypeLPFeeReward, AccountTypeMakerReceivedFeeReward, AccountTypeMakerPaidFeeReward, AccountTypeMarketProposerReward,
		AccountTypeAverageNotionalReward, AccountTypeRelativeReturnReward, AccountTypeValidatorRankingReward, AccountTypeReturnVolatilityReward, AccountTypeRealisedReturnReward, AccountTypeEligibleEntitiesReward, AccountTypeBuyBackFees: /*, AccountTypeLockedForStaking*/
		break
	default:
		return ErrUnsupportedToAccountType
	}

	return nil
}

type GovernanceTransfer struct {
	ID        string // NB: this is the ID of the proposal
	Reference string
	Config    *NewTransferConfiguration
	Status    TransferStatus
	Timestamp time.Time
}

func (g *GovernanceTransfer) IntoProto() *checkpointpb.GovernanceTransfer {
	return &checkpointpb.GovernanceTransfer{
		Id:        g.ID,
		Reference: g.Reference,
		Timestamp: g.Timestamp.UnixNano(),
		Status:    g.Status,
		Config:    g.Config.IntoProto(),
	}
}

func GovernanceTransferFromProto(g *checkpointpb.GovernanceTransfer) *GovernanceTransfer {
	c, _ := NewTransferConfigurationFromProto(g.Config)
	return &GovernanceTransfer{
		ID:        g.Id,
		Reference: g.Reference,
		Timestamp: time.Unix(0, g.Timestamp),
		Status:    g.Status,
		Config:    c,
	}
}

func (g *GovernanceTransfer) IntoEvent(amount *num.Uint, reason, gameID *string) *eventspb.Transfer {
	// Not sure if this symbology gonna work for datanode
	from := "0000000000000000000000000000000000000000000000000000000000000000"
	if len(g.Config.Source) > 0 {
		from = g.Config.Source
	}
	to := g.Config.Destination
	if g.Config.DestinationType == AccountTypeGlobalReward || g.Config.DestinationType == AccountTypeNetworkTreasury || g.Config.DestinationType == AccountTypeGlobalInsurance {
		to = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	out := &eventspb.Transfer{
		Id:              g.ID,
		From:            from,
		FromAccountType: g.Config.SourceType,
		To:              to,
		ToAccountType:   g.Config.DestinationType,
		Asset:           g.Config.Asset,
		Amount:          amount.String(),
		Reference:       g.Reference,
		Status:          g.Status,
		Timestamp:       g.Timestamp.UnixNano(),
		Reason:          reason,
		GameId:          gameID,
	}

	if g.Config.OneOffTransferConfig != nil {
		out.Kind = &eventspb.Transfer_OneOffGovernance{}
		if g.Config.OneOffTransferConfig.DeliverOn > 0 {
			out.Kind = &eventspb.Transfer_OneOffGovernance{
				OneOffGovernance: &eventspb.OneOffGovernanceTransfer{
					DeliverOn: g.Config.OneOffTransferConfig.DeliverOn,
				},
			}
		}
	} else {
		out.Kind = &eventspb.Transfer_RecurringGovernance{
			RecurringGovernance: &eventspb.RecurringGovernanceTransfer{
				StartEpoch:       g.Config.RecurringTransferConfig.StartEpoch,
				EndEpoch:         g.Config.RecurringTransferConfig.EndEpoch,
				DispatchStrategy: g.Config.RecurringTransferConfig.DispatchStrategy,
				Factor:           g.Config.RecurringTransferConfig.Factor,
			},
		}
	}

	return out
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

func (r *RecurringTransfer) IntoEvent(reason *string, gameID *string) *eventspb.Transfer {
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
		GameId:          gameID,
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

func (t *TransferFunds) IntoEvent(reason, gameID *string) *eventspb.Transfer {
	switch t.Kind {
	case TransferCommandKindOneOff:
		return t.OneOff.IntoEvent(reason)
	case TransferCommandKindRecurring:
		return t.Recurring.IntoEvent(reason, gameID)
	default:
		panic("invalid transfer kind")
	}
}

func newTransferBase(id, from string, tf *commandspb.Transfer) (*TransferBase, error) {
	amount, overflowed := num.UintFromString(tf.Amount, 10)
	if overflowed {
		return nil, errors.New("invalid transfer amount")
	}

	tb := &TransferBase{
		ID:              id,
		From:            from,
		FromAccountType: tf.FromAccountType,
		To:              tf.To,
		ToAccountType:   tf.ToAccountType,
		Asset:           tf.Asset,
		Amount:          amount,
		Reference:       tf.Reference,
		Status:          TransferStatusPending,
	}

	if tf.From != nil {
		tb.FromDerivedKey = tf.From
	}

	return tb, nil
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
