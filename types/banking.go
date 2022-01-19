package types

import (
	"errors"
	"fmt"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrMissingTransferKind     = errors.New("missing transfer kind")
	ErrCannotTransferZeroFunds = errors.New("cannot transfer zero funds")
)

type TransferCommandKind int

const (
	TransferCommandKindOneOff TransferCommandKind = iota
	TransferCommandKindRecurring
)

type transferBase struct {
	From            string
	FromAccountType AccountType
	To              string
	ToAccountType   AccountType
	Asset           string
	Amount          *num.Uint
	Reference       string
}

func (t *transferBase) IsValid(okZeroAmount bool) error {
	// ensure amount makes senses
	if !okZeroAmount && t.Amount.IsZero() {
		return ErrCannotTransferZeroFunds
	}

	switch t.FromAccountType {
	case AccountTypeGeneral /*, AccountTypeLockedForStaking*/ :
		break
	default:
		return fmt.Errorf("unsupported from account type: %v", t.FromAccountType)
	}

	switch t.ToAccountType {
	case AccountTypeGeneral, AccountTypeGlobalReward /*, AccountTypeLockedForStaking*/ :
		break
	default:
		return fmt.Errorf("unsupported to account type: %v", t.ToAccountType)
	}

	return nil
}

type OneOffTransfer struct {
	*transferBase
	DeliverOn *time.Time
}

type RecurringTransfer struct {
	*transferBase
	StartEpoch uint64
	EndEpoch   uint64
	Factor     num.Decimal
}

// Just a wrapper, use the Kind on a
// switch to access the proper value
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

func newTransferBase(from string, tf *commandspb.Transfer) (*transferBase, error) {
	amount, overflowed := num.UintFromString(tf.Amount, 10)
	if overflowed {
		return nil, errors.New("invalid transfer amount")
	}

	return &transferBase{
		From:            from,
		FromAccountType: tf.FromAccountType,
		To:              tf.To,
		ToAccountType:   tf.ToAccountType,
		Asset:           tf.Asset,
		Amount:          amount,
		Reference:       tf.Reference,
	}, nil
}

func newOneOffTransfer(base *transferBase, tf *commandspb.Transfer) (*TransferFunds, error) {
	var t *time.Time
	if tf.GetOneOff().GetDeliverOn() > 0 {
		tmpt := time.Unix(tf.GetOneOff().GetDeliverOn(), 0)
		t = &tmpt
	}

	return &TransferFunds{
		Kind: TransferCommandKindOneOff,
		OneOff: &OneOffTransfer{
			transferBase: base,
			DeliverOn:    t,
		},
	}, nil
}

func newRecurringTransfer(base *transferBase, tf *commandspb.Transfer) (*TransferFunds, error) {
	return nil, nil
}
