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

package governance

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
)

func (e *Engine) validateNewProtocolAutomatedPurchaseConfiguration(automatedPurchase *types.NewProtocolAutomatedPurchase, et *enactmentTime, currentTime time.Time) (types.ProposalError, error) {
	if _, ok := e.markets.GetMarket(automatedPurchase.Changes.MarketID, false); !ok {
		return types.ProposalErrorInvalidMarket, ErrMarketDoesNotExist
	}
	if !e.assets.IsEnabled(automatedPurchase.Changes.From) {
		return types.ProposalErrorInvalidAsset, assets.ErrAssetDoesNotExist
	}
	mkt, _ := e.markets.GetMarket(automatedPurchase.Changes.MarketID, false)
	if mkt.GetSpot() == nil {
		return types.ProposalErrorInvalidMarket, fmt.Errorf("market for automated purchase must be a spot market")
	}
	spot := mkt.GetSpot().Spot
	if automatedPurchase.Changes.From != spot.BaseAsset && automatedPurchase.Changes.From != spot.QuoteAsset {
		return types.ProposalErrorInvalidMarket, fmt.Errorf("mismatch between asset for automated purchase and the spot market configuration - asset is not one of base/quote assets of the market")
	}
	if mkt.State == types.MarketStateClosed || mkt.State == types.MarketStateCancelled || mkt.State == types.MarketStateRejected || mkt.State == types.MarketStateTradingTerminated || mkt.State == types.MarketStateSettled {
		return types.ProposalErrorInvalidMarket, fmt.Errorf("market for automated purchase must be active")
	}
	if papConfigured, _ := e.markets.MarketHasActivePAP(automatedPurchase.Changes.MarketID); papConfigured {
		return types.ProposalErrorInvalidMarket, fmt.Errorf("market already has an active protocol automated purchase program")
	}

	tt := automatedPurchase.Changes.AuctionSchedule.GetInternalTimeTriggerSpecConfiguration()
	currentTime = currentTime.Truncate(time.Second)
	if tt.Triggers[0].Initial == nil {
		tt.SetInitial(time.Unix(et.current, 0), currentTime)
	}
	tt.SetNextTrigger(currentTime)

	tt = automatedPurchase.Changes.AuctionVolumeSnapshotSchedule.GetInternalTimeTriggerSpecConfiguration()
	currentTime = currentTime.Truncate(time.Second)
	if tt.Triggers[0].Initial == nil {
		tt.SetInitial(time.Unix(et.current, 0), currentTime)
	}
	tt.SetNextTrigger(currentTime)

	return types.ProposalErrorUnspecified, nil
}
