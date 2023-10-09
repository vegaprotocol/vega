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
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type Delegation struct {
	Party    string
	NodeID   string
	Amount   *num.Uint
	EpochSeq string
}

func DelegationFromProto(d *vega.Delegation) *Delegation {
	amt, _ := num.UintFromString(d.Amount, 10)
	return &Delegation{
		Party:    d.Party,
		NodeID:   d.NodeId,
		Amount:   amt,
		EpochSeq: d.EpochSeq,
	}
}

func (d Delegation) IntoProto() *vega.Delegation {
	return &vega.Delegation{
		Party:    d.Party,
		NodeId:   d.NodeID,
		Amount:   num.UintToString(d.Amount),
		EpochSeq: d.EpochSeq,
	}
}

type Delegate struct {
	NodeID string
	Amount *num.Uint
}

func (d Delegate) IntoProto() *commandspb.DelegateSubmission {
	return &commandspb.DelegateSubmission{
		NodeId: d.NodeID,
		Amount: num.UintToString(d.Amount),
	}
}

func (d Delegate) String() string {
	return fmt.Sprintf(
		"nodeID(%s) amount(%s)",
		d.NodeID,
		stringer.UintPointerToString(d.Amount),
	)
}

type Undelegate struct {
	NodeID string
	Amount *num.Uint
	Method string
}

func (u Undelegate) IntoProto() *commandspb.UndelegateSubmission {
	return &commandspb.UndelegateSubmission{
		NodeId: u.NodeID,
		Amount: num.UintToString(u.Amount),
		Method: commandspb.UndelegateSubmission_Method(commandspb.UndelegateSubmission_Method_value[u.Method]),
	}
}

func (u Undelegate) String() string {
	return fmt.Sprintf(
		"nodeID(%s) amount(%s) method(%s)",
		u.NodeID,
		stringer.UintPointerToString(u.Amount),
		u.Method,
	)
}

// ValidatorData is delegation data for validator.
type ValidatorData struct {
	NodeID            string
	PubKey            string
	StakeByDelegators *num.Uint
	SelfStake         *num.Uint
	Delegators        map[string]*num.Uint
	TmPubKey          string
}

// ValidatorVotingPower is the scaled voting power for the given tm key.
type ValidatorVotingPower struct {
	TmPubKey    string
	VotingPower int64
}

type StakeScoreParams struct {
	MinVal                 num.Decimal
	CompLevel              num.Decimal
	OptimalStakeMultiplier num.Decimal
}
