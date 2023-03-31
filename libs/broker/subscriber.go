package broker

import "code.vegaprotocol.io/vega/core/events"

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/libs/broker Subscriber

// Subscriber interface allows pushing values to subscribers, can be set to
// a Skip state (temporarily not receiving any events), or closed. Otherwise events are pushed.
type Subscriber interface {
	Push(val ...events.Event)
	Skip() <-chan struct{}
	Closed() <-chan struct{}
	C() chan<- []events.Event
	Types() []events.Type
	SetID(id int)
	ID() int
	Ack() bool
}
