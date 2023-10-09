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

package adapters

import (
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
)

func toSpamStatistic(st *apipb.SpamStatistic) *nodetypes.SpamStatistic {
	if st == nil {
		// can happen if pointing to an older version of core where this
		// particular spam statistic doesn't exist yet
		return &nodetypes.SpamStatistic{}
	}
	return &nodetypes.SpamStatistic{
		CountForEpoch: st.CountForEpoch,
		MaxForEpoch:   st.MaxForEpoch,
		BannedUntil:   st.BannedUntil,
	}
}
