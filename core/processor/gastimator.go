package processor

import (
	"context"
	"math"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

const (
	batchFactor       = 0.5
	pegCostFactor     = uint64(50)
	stopCostFactor    = 0.2
	lpShapeCostFactor = uint64(100)
	positionFactor    = uint64(1)
	levelFactor       = 0.1
	high              = 10000
	medium            = 100
	low               = 1
)

type ExecEngine interface {
	GetMarketCounters() map[string]*types.MarketCounters
}

type Gastimator struct {
	minBlockCapacity uint64
	maxGas           uint64
	defaultGas       uint64
	exec             ExecEngine
	marketCounters   map[string]*types.MarketCounters
}

func NewGastimator(exec ExecEngine) *Gastimator {
	return &Gastimator{
		exec:           exec,
		marketCounters: map[string]*types.MarketCounters{},
	}
}

// OnBlockEnd is called at the end of the block to update the per market counters and return the max gas as defined by the network parameter.
func (g *Gastimator) OnBlockEnd() uint64 {
	g.marketCounters = g.exec.GetMarketCounters()
	return g.maxGas
}

// OnMaxGasUpdate updates the max gas from the network parameter.
func (g *Gastimator) OnMinBlockCapacityUpdate(ctx context.Context, minBlockCapacity *num.Uint) error {
	g.minBlockCapacity = minBlockCapacity.Uint64()
	return nil
}

// OnMaxGasUpdate updates the max gas from the network parameter.
func (g *Gastimator) OnMaxGasUpdate(ctx context.Context, max *num.Uint) error {
	g.maxGas = max.Uint64()
	return nil
}

// OnDefaultGasUpdate updates the default gas wanted per transaction.
func (g *Gastimator) OnDefaultGasUpdate(ctx context.Context, def *num.Uint) error {
	g.defaultGas = def.Uint64()
	return nil
}

// GetMaxGas returns the current value of max gas.
func (g *Gastimator) GetMaxGas() uint64 {
	return g.maxGas
}

func (g *Gastimator) GetPriority(tx abci.Tx) uint64 {
	switch tx.Command() {
	case txn.ProposeCommand, txn.VoteCommand:
		return medium
	default:
		if tx.Command().IsValidatorCommand() {
			return high
		}
		return low
	}
}

func (g *Gastimator) CalcGasWantedForTx(tx abci.Tx) (uint64, error) {
	switch tx.Command() {
	case txn.SubmitOrderCommand:
		s := &commandspb.OrderSubmission{}
		if err := tx.Unmarshal(s); err != nil {
			return g.maxGas + 1, err
		}
		return g.orderGastimate(s.MarketId), nil
	case txn.AmendOrderCommand:
		s := &commandspb.OrderAmendment{}
		if err := tx.Unmarshal(s); err != nil {
			return g.maxGas + 1, err
		}
		return g.orderGastimate(s.MarketId), nil
	case txn.CancelOrderCommand:
		s := &commandspb.OrderCancellation{}
		if err := tx.Unmarshal(s); err != nil {
			return g.maxGas + 1, err
		}
		// if it is a cancel for one market
		if len(s.MarketId) > 0 && len(s.OrderId) > 0 {
			return g.cancelOrderGastimate(s.MarketId), nil
		}
		// if it is a cancel for all markets
		return g.defaultGas, nil
	case txn.LiquidityProvisionCommand:
		s := &commandspb.LiquidityProvisionSubmission{}
		if err := tx.Unmarshal(s); err != nil {
			return g.maxGas + 1, err
		}
		return g.lpGastimate(s.MarketId), nil

	case txn.AmendLiquidityProvisionCommand:
		s := &commandspb.LiquidityProvisionAmendment{}
		if err := tx.Unmarshal(s); err != nil {
			return g.maxGas + 1, err
		}
		return g.lpGastimate(s.MarketId), nil
	case txn.CancelLiquidityProvisionCommand:
		s := &commandspb.LiquidityProvisionCancellation{}
		if err := tx.Unmarshal(s); err != nil {
			return g.maxGas + 1, err
		}
		return g.lpGastimate(s.MarketId), nil
	case txn.BatchMarketInstructions:
		s := &commandspb.BatchMarketInstructions{}
		if err := tx.Unmarshal(s); err != nil {
			return g.maxGas + 1, err
		}
		return g.batchGastimate(s), nil
	case txn.StopOrdersSubmissionCommand:
		s := &commandspb.StopOrdersSubmission{}
		if err := tx.Unmarshal(s); err != nil {
			return g.maxGas + 1, err
		}
		var marketId string
		if s.FallsBelow != nil {
			marketId = s.FallsBelow.OrderSubmission.MarketId
		} else {
			marketId = s.RisesAbove.OrderSubmission.MarketId
		}

		return g.orderGastimate(marketId), nil
	case txn.StopOrdersCancellationCommand:
		s := &commandspb.StopOrdersCancellation{}
		if err := tx.Unmarshal(s); err != nil {
			return g.maxGas + 1, err
		}
		// if it is a cancel for one market
		if s.MarketId != nil && s.StopOrderId != nil {
			return g.cancelOrderGastimate(*s.MarketId), nil
		}
		// if it is a cancel for all markets
		return g.defaultGas, nil

	default:
		return g.defaultGas, nil
	}
}

// gasBatch =
// the full cost of the first cancellation (i.e. gasCancel)
// plus batchFactor times sum of all subsequent cancellations added together (each costing gasOrder)
// plus the full cost of the first amendment at gasOrder
// plus batchFactor sum of all subsequent amendments added together (each costing gasOrder)
// plus the full cost of the first limit order at gasOrder
// plus batchFactor sum of all subsequent limit orders added together (each costing gasOrder)
// gasBatch = min(maxGas-1,batchFactor).
func (g *Gastimator) batchGastimate(batch *commandspb.BatchMarketInstructions) uint64 {
	totalBatchGas := 0.0
	for i, os := range batch.Submissions {
		factor := batchFactor
		if i == 0 {
			factor = 1.0
		}
		orderGas := g.orderGastimate(os.MarketId)
		totalBatchGas += factor * float64(orderGas)
	}
	for i, os := range batch.Amendments {
		factor := batchFactor
		if i == 0 {
			factor = 1.0
		}
		orderGas := g.orderGastimate(os.MarketId)
		totalBatchGas += factor * float64(orderGas)
	}
	for i, os := range batch.Cancellations {
		factor := batchFactor
		if i == 0 {
			factor = 1.0
		}
		orderGas := g.cancelOrderGastimate(os.MarketId)
		totalBatchGas += factor * float64(orderGas)
	}
	for i, os := range batch.StopOrdersCancellation {
		factor := batchFactor
		if i == 0 {
			factor = 1.0
		}
		if os.MarketId == nil {
			totalBatchGas += factor * float64(g.defaultGas)
		}
		orderGas := g.cancelOrderGastimate(*os.MarketId)
		totalBatchGas += factor * float64(orderGas)
	}
	for i, os := range batch.StopOrdersSubmission {
		factor := batchFactor
		if i == 0 {
			factor = 1.0
		}
		var marketId string
		if os.FallsBelow != nil {
			marketId = os.FallsBelow.OrderSubmission.MarketId
		} else {
			marketId = os.RisesAbove.OrderSubmission.MarketId
		}
		orderGas := g.orderGastimate(marketId)
		totalBatchGas += factor * float64(orderGas)
	}
	return uint64(math.Min(float64(uint64(totalBatchGas)), float64(g.maxGas-1)))
}

// gasOrder = network.transaction.defaultgas + peg cost factor x pegs
// + LP shape cost factor x shapes
// + position factor x positions
// + level factor x levels
// gasOrder = min(maxGas-1,gasOrder).
func (g *Gastimator) orderGastimate(marketID string) uint64 {
	if marketCounters, ok := g.marketCounters[marketID]; ok {
		return uint64(math.Min(float64(
			g.defaultGas+
				uint64(stopCostFactor*float64(marketCounters.StopOrderCounter))+
				pegCostFactor*marketCounters.PeggedOrderCounter+
				lpShapeCostFactor*marketCounters.LPShapeCount+
				positionFactor*marketCounters.PositionCount+
				uint64(levelFactor*float64(marketCounters.OrderbookLevelCount))),
			math.Max(1.0, float64(g.maxGas/g.minBlockCapacity-1))))
	}
	return g.defaultGas
}

// gasCancel = network.transaction.defaultgas + peg cost factor x pegs
// + LP shape cost factor x shapes
// + level factor x levels
// gasCancel = min(maxGas-1,gasCancel).
func (g *Gastimator) cancelOrderGastimate(marketID string) uint64 {
	if marketCounters, ok := g.marketCounters[marketID]; ok {
		return uint64(math.Min(float64(
			g.defaultGas+
				uint64(stopCostFactor*float64(marketCounters.StopOrderCounter))+
				pegCostFactor*marketCounters.PeggedOrderCounter+
				lpShapeCostFactor*marketCounters.LPShapeCount+
				uint64(0.1*float64(marketCounters.OrderbookLevelCount))),
			math.Max(1.0, float64(g.maxGas/g.minBlockCapacity-1))))
	}
	return g.defaultGas
}

// gasOliq = network.transaction.defaultgas + peg cost factor  x pegs
// + LP shape cost factor x shapes
// + position factor x positions
// + level factor x levels
// gasOliq = min(maxGas-1,gasOliq).
func (g *Gastimator) lpGastimate(marketID string) uint64 {
	if marketCounters, ok := g.marketCounters[marketID]; ok {
		return uint64(math.Min(float64(
			g.defaultGas+
				uint64(stopCostFactor*float64(marketCounters.StopOrderCounter))+
				pegCostFactor*marketCounters.PeggedOrderCounter+
				lpShapeCostFactor*marketCounters.LPShapeCount+
				positionFactor*marketCounters.PositionCount+
				uint64(levelFactor*float64(marketCounters.OrderbookLevelCount))),
			math.Max(1.0, float64(g.maxGas/g.minBlockCapacity-1))))
	}
	return g.defaultGas
}
