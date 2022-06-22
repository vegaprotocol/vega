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

	types "code.vegaprotocol.io/protos/vega"
)

type newMarketResolver VegaResolverRoot

func (r *newMarketResolver) Instrument(ctx context.Context, obj *types.NewMarket) (*types.InstrumentConfiguration, error) {
	return obj.Changes.Instrument, nil
}

func (r *newMarketResolver) DecimalPlaces(ctx context.Context, obj *types.NewMarket) (int, error) {
	return int(obj.Changes.DecimalPlaces), nil
}

func (r *newMarketResolver) PositionDecimalPlaces(ctx context.Context, obj *types.NewMarket) (int, error) {
	return int(obj.Changes.PositionDecimalPlaces), nil
}

func (r *newMarketResolver) RiskParameters(ctx context.Context, obj *types.NewMarket) (RiskModel, error) {
	switch rm := obj.Changes.RiskParameters.(type) {
	case *types.NewMarketConfiguration_LogNormal:
		return rm.LogNormal, nil
	case *types.NewMarketConfiguration_Simple:
		return rm.Simple, nil
	default:
		return nil, errors.New("invalid risk model")
	}
}

func (r *newMarketResolver) Metadata(ctx context.Context, obj *types.NewMarket) ([]string, error) {
	return obj.Changes.Metadata, nil
}

func (r *newMarketResolver) Commitment(ctx context.Context, obj *types.NewMarket) (*types.NewMarketCommitment, error) {
	return obj.LiquidityCommitment, nil
}
