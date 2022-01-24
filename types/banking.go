package types

import (
	"errors"
	"time"

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
	// A stopped transfer.
	TransferStatusStopped TransferStatus = eventspb.Transfer_STATUS_STOPPED
	// A cancelled transfer.
	TransferStatusCancelled TransferStatus = eventspb.Transfer_STATUS_CANCELLED
)

var (
	ErrMissingTransferKind        = errors.New("missing transfer kind")
	ErrCannotTransferZeroFunds    = errors.New("cannot transfer zero funds")
	ErrInvalidFromAccount         = errors.New("invalid from account")
	ErrInvalidToAccount           = errors.New("invalid to account")
	ErrUnsupportedFromAccountType = errors.New("unsupported from account type")
	ErrUnsupportedToAccountType   = errors.New("unsupported to account type")
	ErrEndEpochIsZero             = errors.New("end epoch is zero")
	ErrStartEpochIsZero           = errors.New("start epoch is zero")
	ErrInvalidFactor              = errors.New("invalid factor")
	ErrStartEpochAfterEndEpoch    = errors.New("start epoch after end epoch")
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

func (t *TransferBase) IsValid() error {
	if len(t.From) <= 0 {
		return ErrInvalidFromAccount
	}
	if len(t.To) <= 0 {
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

func (o *OneOffTransfer) IsValid() error {
	if err := o.TransferBase.IsValid(); err != nil {
		return err
	}

	return nil
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
	EndEpoch   *uint64
	Factor     num.Decimal
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

func (t *RecurringTransfer) IntoEvent() *eventspb.Transfer {
	var endEpoch *eventspb.RecurringTransfer_EndEpoch
	if t.EndEpoch != nil {
		endEpoch = &eventspb.RecurringTransfer_EndEpoch{
			EndEpoch: *t.EndEpoch,
		}
	}

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
				EndEpoch_:  endEpoch,
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
	factor, err := num.DecimalFromString(tf.GetRecurring().GetFactor())
	if err != nil {
		return nil, err
	}
	var endEpoch *uint64
	if tf.GetRecurring().EndEpoch_ != nil {
		ee := tf.GetRecurring().GetEndEpoch()
		endEpoch = &ee
	}

	return &TransferFunds{
		Kind: TransferCommandKindRecurring,
		Recurring: &RecurringTransfer{
			TransferBase: base,
			StartEpoch:   tf.GetRecurring().GetStartEpoch(),
			EndEpoch:     endEpoch,
			Factor:       factor,
		},
	}, nil
}

func RecurringTransferFromEvent(p *eventspb.Transfer) *RecurringTransfer {
	var endEpoch *uint64
	if p.GetRecurring().EndEpoch_ != nil {
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
		},
		StartEpoch: p.GetRecurring().GetStartEpoch(),
		EndEpoch:   endEpoch,
		Factor:     factor,
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
