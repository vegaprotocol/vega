// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"context"
	"errors"

	proto "code.vegaprotocol.io/protos/vega"
)

type myTradableInstrumentResolver VegaResolverRoot

func (r *myTradableInstrumentResolver) RiskModel(ctx context.Context, obj *proto.TradableInstrument) (RiskModel, error) {
	switch rm := obj.RiskModel.(type) {
	case *proto.TradableInstrument_LogNormalRiskModel:
		return rm.LogNormalRiskModel, nil
	case *proto.TradableInstrument_SimpleRiskModel:
		return rm.SimpleRiskModel, nil
	default:
		return nil, errors.New("invalid risk model")
	}
}
