package monitor

import "code.vegaprotocol.io/vega/types"

func NewAuctionStateFromSnapshot(mkt *types.Market, as *types.AuctionState) *AuctionState {
	s := AuctionState{
		mode:         as.Mode,
		defMode:      as.DefaultMode,
		trigger:      types.AuctionTriggerOpening,
		end:          as.End,
		start:        as.Start,
		stop:         as.Stop,
		m:            mkt,
		stateChanged: true,
	}

	if as.Begin.IsZero() {
		s.begin = nil
	} else {
		s.begin = &as.Begin
	}

	if as.Extension == types.AuctionTriggerUnspecified {
		s.extension = nil
		s.trigger = types.AuctionTriggerUnspecified
	} else {
		s.extension = &as.Extension
		s.trigger = as.Extension
	}

	return &s
}

func (a AuctionState) Changed() bool {
	return a.stateChanged
}

func (a *AuctionState) GetState() *types.AuctionState {
	as := &types.AuctionState{
		Mode:        a.mode,
		DefaultMode: a.defMode,
		End:         a.end,
		Start:       a.start,
		Stop:        a.stop,
	}
	if a.extension != nil {
		as.Extension = *a.extension
	}

	if a.begin != nil {
		as.Begin = *a.begin
	}

	a.stateChanged = false

	return as
}
