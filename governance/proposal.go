package governance

import types "code.vegaprotocol.io/vega/proto"

// Proposal is a wrapper over Proposal defined in proto interface + payload data
// required to complete proposal enactment
type Proposal struct {
	*types.Proposal
	data proposalPayload
}

type proposalPayload interface{}

// NewMarketData returns new market proposal payload if any
// The payload is data created as first step of the market creation logic in governance.Engine
func (p *Proposal) NewMarketData() *types.Market {
	if market, ok := p.data.(*types.Market); ok {
		return market
	}
	return nil
}
