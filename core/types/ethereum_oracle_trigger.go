package types

import (
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	"golang.org/x/crypto/sha3"
)

type trigger interface {
	isTrigger()
	oneOfProto() interface{} // Calls IntoProto
	GetEthTrigger() *EthCallTrigger
	Hash() []byte
	Trigger(Blockish, Blockish) bool
	String() string
}

type EthTimeTrigger struct {
	Initial uint64
	Every   uint64 // 0 = don't repeat
	Until   uint64 // 0 = forever
}

type Blockish interface {
	NumberU64() uint64
	Time() uint64
}

func (e EthTimeTrigger) Trigger(prev Blockish, current Blockish) bool {
	// Before initial?
	if current.Time() < e.Initial {
		return false
	}

	// Crossing initial boundary?
	if prev.Time() < e.Initial && current.Time() >= e.Initial {
		return true
	}

	// After until?
	if e.Until != 0 && current.Time() > e.Until {
		return false
	}

	if e.Every == 0 {
		return false
	}
	// Somewhere in the middle..
	prevTriggerCount := (prev.Time() - e.Initial) / e.Every
	currentTriggerCount := (current.Time() - e.Initial) / e.Every
	return currentTriggerCount > prevTriggerCount
}

func (e *EthTimeTrigger) isTrigger() {}

func (e *EthTimeTrigger) oneOfProto() interface{} {
	return e.IntoProto()
}

func (e *EthTimeTrigger) IntoProto() *vegapb.EthTimeTrigger {
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

func (e *EthTimeTrigger) GetEthTrigger() *EthCallTrigger {
	return &EthCallTrigger{
		EthTrigger: &EthCallTriggerTime{
			TimeTrigger: e,
		},
	}
}

func (e *EthTimeTrigger) String() string {
	return fmt.Sprintf(
		"initial(%d) every(%d) until(%d)",
		e.Initial,
		e.Every,
		e.Until,
	)
}

func EthTimeTriggerFromProto(protoTrigger *vegapb.EthTimeTrigger) *EthTimeTrigger {

	trigger := &EthTimeTrigger{}
	if protoTrigger != nil {
		trigger.Initial = *protoTrigger.Initial
		trigger.Every = *protoTrigger.Every
		trigger.Until = *protoTrigger.Until
	}

	return trigger
}

func (e *EthTimeTrigger) Hash() []byte {
	hashFunc := sha3.New256()
	ident := fmt.Sprintf("timetrigger: %v/%v/%v", e.Initial, e.Every, e.Until)
	hashFunc.Write([]byte(ident))
	return hashFunc.Sum(nil)
}

type EthCallTriggerTime struct {
	TimeTrigger trigger
}

func (e *EthCallTriggerTime) isTrigger() {}

func (e *EthCallTriggerTime) oneOfProto() interface{} {
	return e.IntoProto()
}

func (e *EthCallTriggerTime) IntoProto() *vegapb.EthCallTrigger_TimeTrigger {
	if e.TimeTrigger != nil {
		switch tp := e.TimeTrigger.(type) {
		case *EthTimeTrigger:
			return &vegapb.EthCallTrigger_TimeTrigger{
				TimeTrigger: tp.IntoProto(),
			}
		}
	}

	return &vegapb.EthCallTrigger_TimeTrigger{}
}

func (e *EthCallTriggerTime) GetEthTrigger() *EthCallTrigger {
	tt := &EthCallTrigger{}
	if e.TimeTrigger != nil {
		switch tp := e.TimeTrigger.(type) {
		case *EthTimeTrigger:
			tt.EthTrigger = &EthCallTriggerTime{
				TimeTrigger: tp,
			}
		}
	}
	return tt
}

func (e *EthCallTriggerTime) String() string {
	ethct := ""
	if e.TimeTrigger != nil {
		switch tp := e.TimeTrigger.(type) {
		case *EthTimeTrigger:
			ethct = tp.String()
		}
	}

	return fmt.Sprintf("ethcalltriggertime(%s)", ethct)
}

func EthCallTriggerTimeFromProto(ett *vegapb.EthCallTrigger_TimeTrigger) *EthCallTriggerTime {
	if ett != nil {
		if ett.TimeTrigger != nil {
			return &EthCallTriggerTime{
				TimeTrigger: EthTimeTriggerFromProto(ett.TimeTrigger),
			}
		}
	}
	return &EthCallTriggerTime{}
}

func (e *EthCallTriggerTime) Hash() []byte {
	if e.TimeTrigger != nil {
		return e.TimeTrigger.Hash()
	}

	// TODO: Error for empty trigger
	return nil
}

func (e *EthCallTriggerTime) Trigger(prev, current Blockish) bool {
	if e.TimeTrigger != nil {
		return e.TimeTrigger.Trigger(prev, current)
	}

	return false
}

type EthCallTrigger struct {
	EthTrigger trigger
}

func (e *EthCallTrigger) isTrigger() {}

func (e *EthCallTrigger) IntoProto() *vegapb.EthCallTrigger {
	if e.Trigger != nil {
		switch tp := e.EthTrigger.(type) {
		case *EthTimeTrigger:
			return &vegapb.EthCallTrigger{
				EthTrigger: &vegapb.EthCallTrigger_TimeTrigger{
					TimeTrigger: tp.IntoProto(),
				},
			}
		}
	}

	// TODO: Return some error?
	/*
		if proto == nil {
			return nil, fmt.Errorf("trigger proto is nil")
		}
	*/
	return &vegapb.EthCallTrigger{}
}

func (e *EthCallTrigger) oneOfProto() interface{} {
	return e.IntoProto()
}

func (e *EthCallTrigger) GetEthTrigger() *EthCallTrigger {
	return e
}

func (e *EthCallTrigger) String() string {
	tt := ""
	if e.Trigger != nil {
		switch tp := e.EthTrigger.(type) {
		case *EthCallTriggerTime:
			if tp.TimeTrigger != nil {
				tt = tp.String()
			}
		}
	}
	return fmt.Sprintf("ethcalltrigger(%s)", tt)
}

func EthCallTriggerFromProto(protoTrigger *vegapb.EthCallTrigger) *EthCallTrigger {
	tr := &EthCallTrigger{}
	if protoTrigger != nil {
		if protoTrigger.EthTrigger != nil {
			switch tp := protoTrigger.EthTrigger.(type) {
			case *vegapb.EthCallTrigger_TimeTrigger:
				tr.EthTrigger = EthCallTriggerTimeFromProto(tp)
			}
		}
	}

	return tr
}

func (e EthCallTrigger) Hash() []byte {
	if e.Trigger != nil {
		return e.EthTrigger.Hash()
	}

	// TODO: Error for empty trigger
	return nil
}

func (e *EthCallTrigger) Trigger(prev, current Blockish) bool {
	if e.EthTrigger != nil {
		return e.EthTrigger.Trigger(prev, current)
	}

	return false
}
