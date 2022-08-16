// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
