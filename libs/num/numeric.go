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

package num

import (
	"errors"
	"fmt"
	"strings"
)

type Numeric struct {
	asInt     *Int
	asUint    *Uint
	asDecimal *Decimal
}

func (n *Numeric) Clone() *Numeric {
	if n.asUint != nil {
		nn := &Numeric{}
		nn.SetUint(n.Uint().Clone())
		return nn
	}

	if n.asInt != nil {
		nn := &Numeric{}
		nn.SetInt(n.Int().Clone())
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
	if n.asInt != nil {
		return n.asInt.String()
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

	if strings.HasPrefix(s, "-") {
		in, _ := IntFromString(s, 10)
		return &Numeric{
			asInt: in,
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

func (n *Numeric) SetInt(in *Int) *Numeric {
	n.asInt = in
	n.asDecimal = nil
	n.asUint = nil
	return n
}

func (n *Numeric) SetUint(u *Uint) *Numeric {
	n.asUint = u
	n.asDecimal = nil
	n.asInt = nil
	return n
}

func (n *Numeric) SetDecimal(d *Decimal) *Numeric {
	n.asDecimal = d
	n.asUint = nil
	n.asInt = nil
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

func (n *Numeric) Int() *Int {
	if n.asInt == nil {
		return nil
	}
	in := *n.asInt
	return &in
}

func (n *Numeric) IsDecimal() bool {
	return n.asDecimal != nil
}

func (n *Numeric) IsUint() bool {
	return n.asUint != nil
}

func (n *Numeric) IsInt() bool {
	return n.asInt != nil
}
