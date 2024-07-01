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

package service_test

import (
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestAMMThings(t *testing.T) {
	mds := getTestMDS(t)

	// lower true pv 702.4119613637248987 l 580723.51752738390596462639919437474617
	// lower false pv 635.3954521864637116 l 610600.1174758454383959875699679680084

	pool := entities.AMMPool{
		ParametersLowerBound:     ptr.From(num.DecimalFromInt64(1800)),
		LowerVirtualLiquidity:    num.DecimalFromFloat(580723.51752738390596462639919437474617),
		LowerTheoreticalPosition: num.DecimalFromFloat(702.4119613637248987),
		ParametersBase:           num.DecimalFromInt64(2000),
		ParametersUpperBound:     ptr.From(num.DecimalFromInt64(2200)),
		UpperVirtualLiquidity:    num.DecimalFromFloat(610600.1174758454383959875699679680084),
		UpperTheoreticalPosition: num.DecimalFromFloat(635.3954521864637116),
	}

	pos := entities.Position{
		OpenVolume: 0,
	}

	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(pos, nil)

	mds.service.ExpandAMM(pool, num.DecimalFromInt64(2000))
	require.False(t, true)
}
