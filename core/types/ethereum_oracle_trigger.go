package types

import (
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type EthCallTrigger interface {
	IntoEthCallTriggerProto() *vegapb.EthCallTrigger
	DeepClone() EthCallTrigger
	String() string
}

func EthCallTriggerFromProto(proto *vegapb.EthCallTrigger) (EthCallTrigger, error) {
	if proto == nil {
		return nil, fmt.Errorf("trigger proto is nil")
	}

	switch t := proto.Trigger.(type) {
	case *vegapb.EthCallTrigger_TimeTrigger:
		return EthTimeTriggerFromProto(t.TimeTrigger), nil
	default:
		return nil, fmt.Errorf("unknown trigger type: %T", proto.Trigger)
	}
}

type EthTimeTrigger struct {
	Initial uint64
	Every   uint64 // 0 = don't repeat
	Until   uint64 // 0 = forever
}

func (e EthTimeTrigger) IntoProto() *vegapb.EthTimeTrigger {
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

func (e EthTimeTrigger) DeepClone() EthCallTrigger {
	return e
}

func (e EthTimeTrigger) IntoEthCallTriggerProto() *vegapb.EthCallTrigger {
	return &vegapb.EthCallTrigger{
		Trigger: &vegapb.EthCallTrigger_TimeTrigger{
			TimeTrigger: e.IntoProto(),
		},
	}
}

func (e EthTimeTrigger) String() string {
	return fmt.Sprintf(
		"initial(%d) every(%d) until(%d)",
		e.Initial,
		e.Every,
		e.Until,
	)
}

func EthTimeTriggerFromProto(protoTrigger *vegapb.EthTimeTrigger) EthTimeTrigger {
	trigger := EthTimeTrigger{}
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
