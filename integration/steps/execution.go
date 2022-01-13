package steps

import (
	"context"

	"code.vegaprotocol.io/vega/types"
)

// the interface for execution engine. The execution engine itself will be wrapped
// so to use it in steps, we'll need to use an interface.
type Execution interface {
	GetMarketData(mktID string) (types.MarketData, error)
	GetMarketState(mktID string) (types.MarketState, error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment, party string) (*types.OrderConfirmation, error)
	CancelOrder(ctx context.Context, cancel *types.OrderCancellation, party string) ([]*types.OrderCancellationConfirmation, error)
	SubmitOrder(ctx context.Context, submission *types.OrderSubmission, party string) (*types.OrderConfirmation, error)
	SubmitLiquidityProvision(ctx context.Context, submission *types.LiquidityProvisionSubmission, party string, lpID string) error
	AmendLiquidityProvision(ctx context.Context, amendment *types.LiquidityProvisionAmendment, party string) error
	CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) error
	SubmitMarket(ctx context.Context, marketConfig *types.Market) error
}
