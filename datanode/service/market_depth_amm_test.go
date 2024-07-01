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

	pool := entities.AMMPool{
		ParametersLowerBound:     ptr.From(num.DecimalFromInt64(100)),
		LowerVirtualLiquidity:    num.DecimalFromInt64(100000),
		LowerTheoreticalPosition: num.DecimalFromInt64(1000),
		ParametersBase:           num.DecimalFromInt64(200),
		ParametersUpperBound:     ptr.From(num.DecimalFromInt64(300)),
		UpperVirtualLiquidity:    num.DecimalFromInt64(100000),
		UpperTheoreticalPosition: num.DecimalFromInt64(1000),
	}

	pos := entities.Position{
		OpenVolume: 0,
	}

	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(pos, nil)

	mds.service.ExpandAMM(pool, num.DecimalFromInt64(200))
	require.False(t, true)
}
