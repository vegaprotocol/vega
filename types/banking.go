package types

import (
	"errors"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
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
	From            string
	FromAccountType AccountType
	To              string
	ToAccountType   AccountType
	Asset           string
	Amount          *num.Uint
	Reference       string
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

type RecurringTransfer struct {
	*TransferBase
	StartEpoch uint64
	EndEpoch   uint64
	Factor     num.Decimal
}

// Just a wrapper, use the Kind on a
// switch to access the proper value.
type TransferFunds struct {
	Kind      TransferCommandKind
	OneOff    *OneOffTransfer
	Recurring *RecurringTransfer
}

func NewTransferFromProto(from string, tf *commandspb.Transfer) (*TransferFunds, error) {
	base, err := newTransferBase(from, tf)
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

func newTransferBase(from string, tf *commandspb.Transfer) (*TransferBase, error) {
	amount, overflowed := num.UintFromString(tf.Amount, 10)
	if overflowed {
		return nil, errors.New("invalid transfer amount")
	}

	return &TransferBase{
		From:            from,
		FromAccountType: tf.FromAccountType,
		To:              tf.To,
		ToAccountType:   tf.ToAccountType,
		Asset:           tf.Asset,
		Amount:          amount,
		Reference:       tf.Reference,
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
