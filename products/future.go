package products

import (
	"time"

	"code.vegaprotocol.io/vega/oracles"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

// Future represent a Future as describe by the market framework
type Future struct {
	SettlementAsset string
	QuoteName       string
	Maturity        time.Time
	Oracle          oracles.Oracle
}

// Settle a position against the future
func (f *Future) Settle(entryPrice uint64, netPosition int64) (*types.FinancialAmount, error) {
	settlementPrice, err := f.Oracle.SettlementPrice()
	if err != nil {
		return nil, err
	}
	return &types.FinancialAmount{
		Asset:  f.SettlementAsset,
		Amount: int64(settlementPrice - entryPrice) * netPosition,
	}, nil
}

// Value - returns the nominal value of a unit given a current mark price
func (f *Future) Value(markPrice uint64) (uint64, error) {
	return markPrice, nil
}

// GetAsset return the asset used by the future
func (f *Future) GetAsset() string {
	return f.SettlementAsset
}

func newFuture(f *types.Future) (*Future, error) {
	maturity, err := time.Parse(time.RFC3339, f.Maturity)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid maturity time format")
	}

	oracle, err := oracles.New(f.Oracle)
	if err != nil {
		return nil, err
	}

	return &Future{
		SettlementAsset: f.SettlementAsset,
		QuoteName:       f.QuoteName,
		Maturity:        maturity,
		Oracle:          oracle,
	}, nil
}
