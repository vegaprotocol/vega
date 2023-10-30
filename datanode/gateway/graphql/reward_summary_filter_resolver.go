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
	"fmt"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type rewardSummaryFilterResolver VegaResolverRoot

func (r *rewardSummaryFilterResolver) FromEpoch(_ context.Context, obj *v2.RewardSummaryFilter, data *int) (err error) {
	obj.FromEpoch, err = intPtrToUint64Ptr(data)
	return
}

func (r *rewardSummaryFilterResolver) ToEpoch(_ context.Context, obj *v2.RewardSummaryFilter, data *int) (err error) {
	obj.ToEpoch, err = intPtrToUint64Ptr(data)
	return
}

func intPtrToUint64Ptr(i *int) (*uint64, error) {
	iVal := ptr.UnBox(i)
	if iVal < 0 {
		return nil, fmt.Errorf("cannot convert to uint - must be positive")
	}
	return ptr.From(uint64(iVal)), nil
}
