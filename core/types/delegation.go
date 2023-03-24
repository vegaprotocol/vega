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
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
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
	Amount *num.Uint
	NodeID string
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
		uintPointerToString(d.Amount),
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
		uintPointerToString(u.Amount),
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
