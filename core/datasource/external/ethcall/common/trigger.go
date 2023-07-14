// Copyright (c) 2023 Gobalsky Labs Limited
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

package common

import (
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type Trigger interface {
	IntoTriggerProto() *vegapb.EthCallTrigger
	DeepClone() Trigger
	String() string
}

func TriggerFromProto(proto *vegapb.EthCallTrigger) (Trigger, error) {
	if proto == nil {
		return nil, fmt.Errorf("trigger proto is nil")
	}

	switch t := proto.Trigger.(type) {
	case *vegapb.EthCallTrigger_TimeTrigger:
		return TimeTriggerFromProto(t.TimeTrigger), nil
	default:
		return nil, fmt.Errorf("unknown trigger type: %T", proto.Trigger)
	}
}

type TimeTrigger struct {
	Initial uint64
	Every   uint64 // 0 = don't repeat
	Until   uint64 // 0 = forever
}

func (e TimeTrigger) IntoProto() *vegapb.EthTimeTrigger {
	var initial, every, until *uint64

	if e.Initial != 0 {
		initial = &e.Initial
	}

	if e.Every != 0 {
		every = &e.Every
	}

	if e.Until != 0 {
		until = &e.Until
	}

	return &vegapb.EthTimeTrigger{
		Initial: initial,
		Every:   every,
		Until:   until,
	}
}

func (e TimeTrigger) DeepClone() Trigger {
	return e
}

func (e TimeTrigger) IntoTriggerProto() *vegapb.EthCallTrigger {
	return &vegapb.EthCallTrigger{
		Trigger: &vegapb.EthCallTrigger_TimeTrigger{
			TimeTrigger: e.IntoProto(),
		},
	}
}

func (e TimeTrigger) String() string {
	return fmt.Sprintf(
		"initial(%d) every(%d) until(%d)",
		e.Initial,
		e.Every,
		e.Until,
	)
}

func TimeTriggerFromProto(protoTrigger *vegapb.EthTimeTrigger) TimeTrigger {
	trigger := TimeTrigger{}
	if protoTrigger == nil {
		return trigger
	}

	if protoTrigger.Initial != nil {
		trigger.Initial = *protoTrigger.Initial
	}
	if protoTrigger.Every != nil {
		trigger.Every = *protoTrigger.Every
	}
	if protoTrigger.Until != nil {
		trigger.Until = *protoTrigger.Until
	}
	return trigger
}
