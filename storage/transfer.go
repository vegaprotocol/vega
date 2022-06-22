// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package storage

import (
	"sort"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type Transfers struct {
	cfg Config
	log *logging.Logger

	mu sync.RWMutex
	// all transfers
	transfers map[string]eventspb.Transfer
	// mapping from pubkey -> tranfer id set
	froms map[string]map[string]struct{}
	// mapping to pubkey -> transfer id set
	tos map[string]map[string]struct{}
}

func NewTransfers(
	log *logging.Logger,
	cfg Config,
) *Transfers {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Transfers{
		log:       log,
		cfg:       cfg,
		transfers: map[string]eventspb.Transfer{},
		froms:     map[string]map[string]struct{}{},
		tos:       map[string]map[string]struct{}{},
	}
}

// ReloadConf update the internal conf of the market
func (s *Transfers) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.cfg = cfg
}

func (t *Transfers) AddTransfer(tf eventspb.Transfer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.transfers[tf.Id]
	if ok {
		// this transfers already exists, no need to
		// add it again in the mappings tables
		// this is a status update
		t.transfers[tf.Id] = tf
		return
	}

	// doesn't exist, add it to all tables
	t.transfers[tf.Id] = tf

	froms, ok := t.froms[tf.From]
	if !ok {
		froms = map[string]struct{}{}
		t.froms[tf.From] = froms
	}
	froms[tf.Id] = struct{}{}

	tos, ok := t.tos[tf.To]
	if !ok {
		tos = map[string]struct{}{}
		t.tos[tf.To] = tos
	}
	tos[tf.To] = struct{}{}
}

func (t *Transfers) GetAll(
	pubkey string,
	isFrom, isTo bool,
) []*eventspb.Transfer {
	t.mu.RLock()
	defer t.mu.RUnlock()
	transferIDs := map[string]struct{}{}

	if isFrom {
		if froms, ok := t.froms[pubkey]; ok {
			for k := range froms {
				transferIDs[k] = struct{}{}
			}
		}
	}

	if isTo {
		if tos, ok := t.tos[pubkey]; ok {
			for k := range tos {
				transferIDs[k] = struct{}{}
			}
		}
	}

	if !isTo && !isFrom {
		for id := range t.transfers {
			transferIDs[id] = struct{}{}
		}
	}

	out := make([]*eventspb.Transfer, 0, len(transferIDs))
	for k := range transferIDs {
		tf := t.transfers[k]
		out = append(out, &tf)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp < out[j].Timestamp })

	return out
}
