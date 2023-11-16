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

package entities

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

type RiskFactor struct {
	MarketID MarketID
	Short    decimal.Decimal
	Long     decimal.Decimal
	TxHash   TxHash
	VegaTime time.Time
}

func RiskFactorFromProto(factor *vega.RiskFactor, txHash TxHash, vegaTime time.Time) (*RiskFactor, error) {
	var short, long decimal.Decimal
	var err error

	if short, err = decimal.NewFromString(factor.Short); err != nil {
		return nil, fmt.Errorf("invalid value for short: %s - %v", factor.Short, err)
	}

	if long, err = decimal.NewFromString(factor.Long); err != nil {
		return nil, fmt.Errorf("invalid value for long: %s - %v", factor.Long, err)
	}

	return &RiskFactor{
		MarketID: MarketID(factor.Market),
		Short:    short,
		Long:     long,
		TxHash:   txHash,
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
