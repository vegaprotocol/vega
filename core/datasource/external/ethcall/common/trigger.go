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
