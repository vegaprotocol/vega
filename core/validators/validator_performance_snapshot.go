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

package validators

import (
	"sort"

	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func (vp *validatorPerformance) Deserialize(proto *v1.ValidatorPerformance) {
	if proto != nil {
		tot := int64(0)
		for _, stats := range proto.ValidatorPerfStats {
			tot += int64(stats.Proposed)
			vp.proposals[stats.ValidatorAddress] = int64(stats.Proposed)
		}
		vp.total = tot
	}
}

func (vp *validatorPerformance) Serialize() *v1.ValidatorPerformance {
	keys := make([]string, 0, len(vp.proposals))
	for k := range vp.proposals {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	stats := make([]*v1.PerformanceStats, 0, len(vp.proposals))
	for _, addr := range keys {
		stat := vp.proposals[addr]
		stats = append(stats, &v1.PerformanceStats{
			ValidatorAddress: addr,
			Proposed:         uint64(stat),
		})
	}

	return &v1.ValidatorPerformance{
		ValidatorPerfStats: stats,
	}
}
