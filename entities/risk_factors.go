package entities

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/protos/vega"

	"github.com/shopspring/decimal"
)

type RiskFactor struct {
	MarketID MarketID
	Short    decimal.Decimal
	Long     decimal.Decimal
	VegaTime time.Time
}

func RiskFactorFromProto(factor *vega.RiskFactor, vegaTime time.Time) (*RiskFactor, error) {
	var short, long decimal.Decimal
	var err error

	if short, err = decimal.NewFromString(factor.Short); err != nil {
		return nil, fmt.Errorf("invalid value for short: %s - %v", factor.Short, err)
	}

	if long, err = decimal.NewFromString(factor.Long); err != nil {
		return nil, fmt.Errorf("invalid value for long: %s - %v", factor.Long, err)
	}

	return &RiskFactor{
		MarketID: NewMarketID(factor.Market),
		Short:    short,
		Long:     long,
		VegaTime: vegaTime,
	}, nil
}

func (rf *RiskFactor) ToProto() *vega.RiskFactor {
	return &vega.RiskFactor{
		Market: rf.MarketID.String(),
		Short:  rf.Short.String(),
		Long:   rf.Long.String(),
	}
}
