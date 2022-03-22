package entities

import (
	"encoding/hex"
	"fmt"
	"time"

	"code.vegaprotocol.io/protos/vega"

	"github.com/shopspring/decimal"
)

type RiskFactor struct {
	MarketID []byte
	Short    decimal.Decimal
	Long     decimal.Decimal
	VegaTime time.Time
}

func RiskFactorFromProto(factor *vega.RiskFactor, vegaTime time.Time) (*RiskFactor, error) {
	id, err := makeID(factor.Market)
	if err != nil {
		return nil, fmt.Errorf("invalid market id: %w", err)
	}

	var short, long decimal.Decimal
	if short, err = decimal.NewFromString(factor.Short); err != nil {
		return nil, fmt.Errorf("invalid value for short: %s - %v", factor.Short, err)
	}

	if long, err = decimal.NewFromString(factor.Long); err != nil {
		return nil, fmt.Errorf("invalid value for long: %s - %v", factor.Long, err)
	}

	return &RiskFactor{
		MarketID: id,
		Short:    short,
		Long:     long,
		VegaTime: vegaTime,
	}, nil
}

func (rf *RiskFactor) ToProto() *vega.RiskFactor {
	id := hex.EncodeToString(rf.MarketID)
	return &vega.RiskFactor{
		Market: id,
		Short:  rf.Short.String(),
		Long:   rf.Long.String(),
	}
}
