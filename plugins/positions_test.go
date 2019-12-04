package plugins_test

import "code.vegaprotocol.io/vega/events"

type posStub struct {
	ch chan []events.SettlePosition
	k  int
}

func (p posStub) Subscribe() (<-chan []events.SettlePosition, int) {
	return p.ch, p.k
}

func (p *posStub) Unsubscribe(_ int) {
	// ensure the subscribed channel is closed
	close(p.ch)
	p.ch = make(chan []events.SettlePosition, 1)
}
