package validators

import (
	v1 "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"sort"
)

func (vp *validatorPerformance) deserialize(proto *v1.ValidatorPerformance) {

	if proto != nil {
		for _, stats := range proto.ValidatorPerfStats {
			vp.performance[stats.ValidatorAddress] = &performanceStats{
				proposed:           stats.Proposed,
				elected:            stats.Elected,
				voted:              stats.Voted,
				lastHeightVoted:    stats.LastHeightVoted,
				lastHeightProposed: stats.LastHeightProposed,
				lastHeightElected:  stats.LastHeightElected,
			}
		}
	}
}

func (vp *validatorPerformance) serialize() *v1.ValidatorPerformance {

	keys := make([]string, 0, len(vp.performance))
	for k := range vp.performance {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	stats := make([]*v1.PerformanceStats, 0, len(vp.performance))
	for _, addr := range keys {
		stat := vp.performance[addr]
		stats = append(stats, &v1.PerformanceStats{
			ValidatorAddress:   addr,
			Proposed:           stat.proposed,
			Elected:            stat.elected,
			Voted:              stat.voted,
			LastHeightVoted:    stat.lastHeightVoted,
			LastHeightProposed: stat.lastHeightProposed,
			LastHeightElected:  stat.lastHeightElected,
		})
	}

	return &v1.ValidatorPerformance{
		ValidatorPerfStats: stats,
	}
}
