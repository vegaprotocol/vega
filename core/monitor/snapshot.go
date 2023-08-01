// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
