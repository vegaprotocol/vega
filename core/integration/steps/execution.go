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

package steps

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/types"
)

// the interface for execution engine. The execution engine itself will be wrapped
// so to use it in steps, we'll need to use an interface.
type Execution interface {
	GetMarketData(mktID string) (types.MarketData, error)
	GetMarketState(mktID string) (types.MarketState, error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment, party string) (*types.OrderConfirmation, error)
	CancelOrder(ctx context.Context, cancel *types.OrderCancellation, party string) ([]*types.OrderCancellationConfirmation, error)
	SubmitOrder(ctx context.Context, submission *types.OrderSubmission, party string) (*types.OrderConfirmation, error)
	SubmitLiquidityProvision(ctx context.Context, submission *types.LiquidityProvisionSubmission, party string, lpID string,
		deterministicID string) error
	AmendLiquidityProvision(ctx context.Context, amendment *types.LiquidityProvisionAmendment, party string) error
	CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) error
	SubmitMarket(ctx context.Context, marketConfig *types.Market, proposer string, oos time.Time) error
	StartOpeningAuction(ctx context.Context, marketID string) error
	UpdateMarket(ctx context.Context, marketConfig *types.Market) error
	BlockEnd(ctx context.Context)
	GetMarket(parentID string, settled bool) (types.Market, bool)
	SucceedMarket(ctx context.Context, successor, parent string) error
}
