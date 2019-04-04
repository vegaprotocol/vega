package products

import (
	"time"

	"code.vegaprotocol.io/vega/internal/oracles"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

type Future struct {
	Asset    string
	Maturity time.Time
	Oracle   oracles.Oracle
}

func (f *Future) Settle(entryPrice uint64, netPosition uint64) (*FinancialAmount, error) {
	settlementPrice, err := f.Oracle.SettlementPrice()
	if err != nil {
		return nil, err
	}
	return &FinancialAmount{
		Asset:  f.Asset,
		Amount: (settlementPrice - entryPrice) * netPosition,
	}, nil
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
		Asset:    f.Asset,
		Maturity: maturity,
		Oracle:   oracle,
	}, nil
}
