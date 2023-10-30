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

package gql

import (
	"context"

	v1 "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type fundingPeriodResolver VegaResolverRoot

func (r *fundingPeriodResolver) Seq(ctx context.Context, obj *v1.FundingPeriod) (int, error) {
	return int(obj.Seq), nil
}

func (r *fundingPeriodResolver) StartTime(ctx context.Context, obj *v1.FundingPeriod) (int64, error) {
	return obj.Start, nil
}

func (r *fundingPeriodResolver) EndTime(ctx context.Context, obj *v1.FundingPeriod) (*int64, error) {
	return obj.End, nil
}

type fundingPeriodDataPointResolver VegaResolverRoot

func (r *fundingPeriodDataPointResolver) Seq(ctx context.Context, obj *v1.FundingPeriodDataPoint) (int, error) {
	return int(obj.Seq), nil
}

func (r *fundingPeriodDataPointResolver) DataPointSource(ctx context.Context, obj *v1.FundingPeriodDataPoint) (*v1.FundingPeriodDataPoint_Source, error) {
	return &obj.DataPointType, nil
}
