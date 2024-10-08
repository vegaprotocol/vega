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
	"encoding/hex"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
)

type treeKeyToSnapshot struct {
	treeKey   []byte
	namespace types.SnapshotNamespace
}

type snapshotResult struct {
	input    treeKeyToSnapshot
	toRemove bool
	err      error
	state    []byte
	updated  bool
}

func gatherState(e *Engine, treeKeysToSnapshotChan chan treeKeyToSnapshot, snapshotResultsChan chan<- snapshotResult, treeKeysCounter *atomic.Int64, recordMetrics *snapMetricsState) {
	for toSnapshot := range treeKeysToSnapshotChan {
		// this is for metrics
		startTime := time.Now()
		currentNamespace := toSnapshot.namespace

		treeKeyStr := string(toSnapshot.treeKey)

		t0 := time.Now()

		e.snapshotLock.RLock()
		provider := e.treeKeysToProviders[treeKeyStr]
		providerKey := e.treeKeysToProviderKeys[treeKeyStr]
		e.snapshotLock.RUnlock()

		if provider.Stopped() {
			snapshotResultsChan <- snapshotResult{input: toSnapshot, updated: true, toRemove: true}
			if treeKeysCounter.Add(-1) <= 0 {
				close(treeKeysToSnapshotChan)
				close(snapshotResultsChan)
			}
			continue
		}

		state, additionalProviders, err := provider.GetState(providerKey)
		if err != nil {
			snapshotResultsChan <- snapshotResult{input: toSnapshot, err: err, updated: true}
			close(treeKeysToSnapshotChan)
			close(snapshotResultsChan)
			return
		}

		var treeKeys [][]byte
		var ok bool
		additionalTreeKeysToSnapshot := []treeKeyToSnapshot{}

		// The provider has generated new providers, register them with the engine
		// add them to the AVL tree
		for _, additionalProvider := range additionalProviders {
			knownTreeKeys := map[string]struct{}{}
			// need to atomically check what's in there for the tree key and then add the provider
			e.snapshotLock.Lock()
			treeKeys, ok = e.namespacesToTreeKeys[additionalProvider.Namespace()]
			if ok {
				for _, treeKey := range treeKeys {
					knownTreeKeys[string(treeKey)] = struct{}{}
				}
			}
			e.addProviders(additionalProvider)
			e.snapshotLock.Unlock()
			e.log.Debug("Additional provider added by the worker",
				logging.String("namespace", additionalProvider.Namespace().String()),
			)

			e.snapshotLock.RLock()
			treeKeys, ok = e.namespacesToTreeKeys[additionalProvider.Namespace()]
			e.snapshotLock.RUnlock()
			if !ok || len(treeKeys) == 0 {
				continue
			}

			for _, tk := range treeKeys {
				// ignore tree keys we've already done
				if _, ok := knownTreeKeys[string(tk)]; ok {
					continue
				}
				additionalTreeKeysToSnapshot = append(additionalTreeKeysToSnapshot, treeKeyToSnapshot{treeKey: tk, namespace: additionalProvider.Namespace()})
			}
		}

		e.log.Debug("State updated",
			logging.String("tree-key", treeKeyStr),
			logging.String("hash", hex.EncodeToString(crypto.Hash(state))),
			logging.Float64("took", time.Since(t0).Seconds()),
		)

		treeKeysCounter.Add(int64(len(additionalTreeKeysToSnapshot)))

		for _, treeKeyToSnapshot := range additionalTreeKeysToSnapshot {
			treeKeysToSnapshotChan <- treeKeyToSnapshot
		}

		snapshotResultsChan <- snapshotResult{input: toSnapshot, state: state, updated: true}

		if treeKeysCounter.Add(-1) <= 0 {
			close(treeKeysToSnapshotChan)
			close(snapshotResultsChan)
		}

		timeTaken := time.Since(startTime)
		recordMetrics.Register(currentNamespace.String(), timeTaken, len(state))
	}
}
