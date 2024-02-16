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

package steps

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

// the interface for execution engine. The execution engine itself will be wrapped
// so to use it in steps, we'll need to use an interface.
type Execution interface {
	GetMarketData(mktID string) (types.MarketData, error)
	GetMarketState(mktID string) (types.MarketState, error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment, party string) (*types.OrderConfirmation, error)
	CancelOrder(ctx context.Context, cancel *types.OrderCancellation, party string) ([]*types.OrderCancellationConfirmation, error)
	CancelStopOrder(ctx context.Context, cancel *types.StopOrdersCancellation, party string) error
	SubmitOrder(ctx context.Context, submission *types.OrderSubmission, party string) (*types.OrderConfirmation, error)
	SubmitStopOrder(ctx context.Context, submission *types.StopOrdersSubmission, party string) (*types.OrderConfirmation, error)
	SubmitLiquidityProvision(ctx context.Context, submission *types.LiquidityProvisionSubmission, party string, lpID string,
		deterministicID string) error
	AmendLiquidityProvision(ctx context.Context, amendment *types.LiquidityProvisionAmendment, party string) error
	CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) error
	SubmitSpotMarket(ctx context.Context, marketConfig *types.Market, proposer string, oos time.Time) error
	SubmitMarket(ctx context.Context, marketConfig *types.Market, proposer string, oos time.Time) error
	StartOpeningAuction(ctx context.Context, marketID string) error
	UpdateMarket(ctx context.Context, marketConfig *types.Market) error
	UpdateSpotMarket(ctx context.Context, marketConfig *types.Market) error
	BlockEnd(ctx context.Context)
	GetMarket(parentID string, settled bool) (types.Market, bool)
	SucceedMarket(ctx context.Context, successor, parent string) error

	// even though the batch processing is done above the execution engine, from the feature test point of view
	// it is part of the execution engine
	StartBatch(party string) error
	AddSubmitOrderToBatch(submission *types.OrderSubmission, party string) error
	ProcessBatch(ctx context.Context, party string) error
	OnEpochEvent(ctx context.Context, epoch types.Epoch)
	UpdateMarketState(ctx context.Context, changes *types.MarketStateUpdateConfiguration) error
	UpdateMarginMode(ctx context.Context, party, marketID string, marginMode types.MarginMode, marginFactor num.Decimal) error

	// AMM stuff
	SubmitAMM(ctx context.Context, submit *types.SubmitAMM) error
	AmendAMM(ctx context.Context, submit *types.AmendAMM) error
	CancelAMM(ctx context.Context, cancel *types.CancelAMM) error
}
