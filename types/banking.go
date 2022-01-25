package types

import (
	"errors"
	"time"

	vegapb "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
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
)

var (
	ErrMissingTransferKind        = errors.New("missing transfer kind")
	ErrCannotTransferZeroFunds    = errors.New("cannot transfer zero funds")
	ErrInvalidFromAccount         = errors.New("invalid from account")
	ErrInvalidToAccount           = errors.New("invalid to account")
	ErrUnsupportedFromAccountType = errors.New("unsupported from account type")
	ErrUnsupportedToAccountType   = errors.New("unsupported to account type")
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
}

func (t *TransferBase) IsValid(okZeroAmount bool) error {
	if len(t.From) <= 0 {
		return ErrInvalidFromAccount
	}
	if len(t.To) <= 0 {
		return ErrInvalidToAccount
	}

	// ensure amount makes senses
	if !okZeroAmount && t.Amount.IsZero() {
		return ErrCannotTransferZeroFunds
	}

	switch t.FromAccountType {
	case AccountTypeGeneral /*, AccountTypeLockedForStaking*/ :
		break
	default:
		return ErrUnsupportedFromAccountType
	}

	switch t.ToAccountType {
	case AccountTypeGeneral, AccountTypeGlobalReward /*, AccountTypeLockedForStaking*/ :
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

func OneOffTransferFromEvent(p *eventspb.Transfer) *OneOffTransfer {
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
		},
		DeliverOn: deliverOn,
	}
}

func (t *OneOffTransfer) IntoEvent() *eventspb.Transfer {
	out := &eventspb.Transfer{
		Id:              t.ID,
		From:            t.From,
		FromAccountType: t.FromAccountType,
		To:              t.To,
		ToAccountType:   t.ToAccountType,
		Asset:           t.Asset,
		Amount:          t.Amount.String(),
		Reference:       t.Reference,
		Status:          t.Status,
	}

	if t.DeliverOn != nil {
		out.Kind = &eventspb.Transfer_OneOff{
			OneOff: &eventspb.OneOffTransfer{
				DeliverOn: t.DeliverOn.Unix(),
			},
		}
	}

	return out
}

type RecurringTransfer struct {
	*TransferBase
	StartEpoch uint64
	EndEpoch   uint64
	Factor     num.Decimal
}

func (t *RecurringTransfer) IntoEvent() *eventspb.Transfer {
	return &eventspb.Transfer{
		Id:              t.ID,
		From:            t.From,
		FromAccountType: t.FromAccountType,
		To:              t.To,
		ToAccountType:   t.ToAccountType,
		Asset:           t.Asset,
		Amount:          t.Amount.String(),
		Reference:       t.Reference,
		Status:          t.Status,
		Kind: &eventspb.Transfer_Recurring{
			Recurring: &eventspb.RecurringTransfer{
				StartEpoch: t.StartEpoch,
				EndEpoch:   &vegapb.Uint64Value{Value: t.EndEpoch},
				Factor:     t.Factor.String(),
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

func (t *TransferFunds) IntoEvent() *eventspb.Transfer {
	switch t.Kind {
	case TransferCommandKindOneOff:
		return t.OneOff.IntoEvent()
	case TransferCommandKindRecurring:
		return t.Recurring.IntoEvent()
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
		tmpt := time.Unix(tf.GetOneOff().GetDeliverOn(), 0)
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
	return nil, nil
}
