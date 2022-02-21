package validators

import (
	"sort"

	v1 "code.vegaprotocol.io/protos/vega/snapshot/v1"
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
