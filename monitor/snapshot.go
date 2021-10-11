package monitor

import "code.vegaprotocol.io/vega/types"

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

func (a *AuctionState) RestoreState(as *types.AuctionState) {
	a.mode = as.Mode
	a.defMode = as.DefaultMode
	a.begin = &as.Begin
	a.end = as.End
	a.start = as.Start
	a.stop = as.Stop
	a.extension = &as.Extension

}
