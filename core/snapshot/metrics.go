package snapshot

import (
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/metrics"
	"golang.org/x/exp/maps"
)

type snapMetrics struct {
	timeTaken time.Duration
	size      int
}

type snapMetricsState struct {
	namespaces map[string]snapMetrics
	mtx        sync.Mutex
}

func newSnapMetricsState() *snapMetricsState {
	return &snapMetricsState{
		namespaces: map[string]snapMetrics{},
	}
}

func (s *snapMetricsState) Register(
	namespace string,
	timeTaken time.Duration,
	size int,
) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	sm := s.namespaces[namespace]
	sm.size += size
	sm.timeTaken += timeTaken
	s.namespaces[namespace] = sm
}

func (s *snapMetricsState) Report(blockHeight uint64) {
	namespaces := maps.Keys(s.namespaces)
	sort.Strings(namespaces)

	for _, v := range namespaces {
		stat := s.namespaces[v]
		metrics.RegisterSnapshotNamespaces(v, stat.timeTaken, stat.size)
	}

	metrics.RegisterSnapshotBlockHeight(blockHeight)
}
