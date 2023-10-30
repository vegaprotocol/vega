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

package node

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

func BuildRoundRobinSelectorWithRetryingNodes(log *zap.Logger, hosts []string, retries uint64, requestTTL time.Duration) (Selector, error) {
	nodes := make([]Node, 0, len(hosts))
	for _, host := range hosts {
		n, err := NewRetryingNode(log.Named("retrying-node"), host, retries, requestTTL)
		if err != nil {
			return nil, fmt.Errorf("could not initialize the node %q: %w", host, err)
		}
		nodes = append(nodes, n)
	}

	nodeSelector, err := NewRoundRobinSelector(log.Named("round-robin-selector"), nodes...)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate the round-robin node selector: %w", err)
	}

	return nodeSelector, nil
}
