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

package monitor

import (
	"code.vegaprotocol.io/vega/core/types"
)

func NewAuctionStateFromSnapshot(mkt *types.Market, as *types.AuctionState) *AuctionState {
	s := AuctionState{
		mode:               as.Mode,
		defMode:            as.DefaultMode,
		trigger:            as.Trigger,
		end:                as.End,
		start:              as.Start,
		stop:               as.Stop,
		m:                  mkt,
		extensionEventSent: as.ExtensionEventSent,
	}

	if as.Begin.IsZero() {
		s.begin = nil
	} else {
		s.begin = &as.Begin
	}

	if as.Extension == types.AuctionTriggerUnspecified {
		s.extension = nil
	} else {
		s.extension = &as.Extension
	}
	return &s
}

func (a AuctionState) Changed() bool {
	return true
}

func (a *AuctionState) GetState() *types.AuctionState {
	as := &types.AuctionState{
		Mode:               a.mode,
		DefaultMode:        a.defMode,
		End:                a.end,
		Start:              a.start,
		Stop:               a.stop,
		Trigger:            a.trigger,
		ExtensionEventSent: a.extensionEventSent,
	}
	if a.extension != nil {
		as.Extension = *a.extension
	}

	if a.begin != nil {
		as.Begin = *a.begin
	}

	return as
}
