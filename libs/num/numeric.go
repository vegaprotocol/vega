package num

import (
	"errors"
	"fmt"
	"strings"
)

type Numeric struct {
	asUint    *Uint
	asDecimal *Decimal
}

func (n *Numeric) Clone() *Numeric {
	if n.asUint != nil {
		nn := n
		return nn
	}

	nn := &Numeric{}

	if n.asDecimal != nil {
		decimal := *n.asDecimal
		nn.asDecimal = &decimal
	}
	return nn
}

func (n *Numeric) String() string {
	if n.asUint != nil {
		return n.asUint.String()
	}
	if n.asDecimal != nil {
		return n.asDecimal.String()
	}

	return ""
}

func NumericToString(n *Numeric) string {
	if n == nil {
		return ""
	}

	return n.String()
}

func NumericFromString(s string) (*Numeric, error) {
	if s == "" {
		return nil, nil
	}

	// Check if the provided string contains a ".", because if it does not,
	// the DecimalFromString will return it as int
	split := strings.Split(s, ".")
	if len(split) > 1 {
		d, err := DecimalFromString(s)
		if err != nil {
			return nil, fmt.Errorf("error obtaining decimal from string: %s", err.Error())
		}
		return &Numeric{
			asDecimal: &d,
		}, nil
	}

	u, _ := UintFromString(s, 10)

	return &Numeric{
		asUint: u,
	}, nil
}

// ScaleTo calculates the current contained value - decimal or uint - scaled to the target decimals.
func (n *Numeric) ScaleTo(op, tdp int64) (*Uint, error) {
	base := DecimalFromInt64(10)
	if n.asDecimal != nil {
		scaled := n.asDecimal.Mul(base.Pow(DecimalFromInt64(tdp)))
		r, overflow := UintFromDecimal(scaled)
		if overflow {
			return nil, errors.New("failed to scale settlement data, overflow occurred")
		}
		return r, nil
	}

	if n.asUint != nil {
		scaled := base.Pow(DecimalFromInt64(tdp - op))
		r, overflow := UintFromDecimal(n.asUint.ToDecimal().Mul(scaled))
		if overflow {
			return nil, errors.New("failed to scale settlement data, overflow occurred")
		}
		return r, nil
	}

	return nil, nil
}

func (n *Numeric) SupportDecimalPlaces(dp int64) bool {
	if n.IsDecimal() {
		decimalParts := strings.Split(n.Decimal().String(), ".")
		if len(decimalParts) > 1 {
			if int64(len(decimalParts[1])) > dp {
				return false
			}
		}
	}

	return true
}

func (n *Numeric) SetUint(u *Uint) *Numeric {
	n.asUint = u
	n.asDecimal = nil
	return n
}

func (n *Numeric) SetDecimal(d *Decimal) *Numeric {
	n.asDecimal = d
	n.asUint = nil

	return n
}

func (n *Numeric) Decimal() *Decimal {
	if n.asDecimal == nil {
		return nil
	}
	d := *n.asDecimal
	return &d
}

func (n *Numeric) Uint() *Uint {
	if n.asUint == nil {
		return nil
	}
	u := *n.asUint
	return &u
}

func (n *Numeric) IsDecimal() bool {
	return n.asDecimal != nil
}

func (n *Numeric) IsUint() bool {
	return n.asUint != nil
}
