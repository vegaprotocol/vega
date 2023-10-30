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
