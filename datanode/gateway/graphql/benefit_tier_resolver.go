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
	"strconv"

	"code.vegaprotocol.io/vega/protos/vega"
)

type benefitTierResolver VegaResolverRoot

func (br *benefitTierResolver) MinimumEpochs(_ context.Context, obj *vega.BenefitTier) (int, error) {
	minEpochs, err := strconv.ParseInt(obj.MinimumEpochs, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse minimum epochs %s: %v", obj.MinimumEpochs, err)
	}

	return int(minEpochs), nil
}
