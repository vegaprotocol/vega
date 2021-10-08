package monitor

import "code.vegaprotocol.io/vega/types"

func (a AuctionState) Changed() bool {
	return a.stateChanged
}

func (a *AuctionState) GetState() *types.AuctionState {
	as := &types.AuctionState{
		Mode:        a.mode,
		DefaultMode: a.defMode,
		Begin:       *a.begin,
		End:         a.end,
		Start:       a.start,
		Stop:        a.stop,
		Extension:   *a.extension,
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
