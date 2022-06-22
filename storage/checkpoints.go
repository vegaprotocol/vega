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
	"fmt"

	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"github.com/dgraph-io/badger/v2"
	"github.com/golang/protobuf/proto"
)

type Checkpoints struct {
	Config
	badger          *badgerStore
	log             *logging.Logger
	onCriticalError func()
}

func NewCheckpoints(log *logging.Logger, home string, c Config, onCriticalError func()) (*Checkpoints, error) {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	db, err := newBadgerStore(getOptionsFromConfig(c.Checkpoints, home, log))
	if err != nil {
		return nil, fmt.Errorf("couldn't open Badger checkpoints database: %w", err)
	}
	return &Checkpoints{
		Config:          c,
		badger:          db,
		log:             log,
		onCriticalError: onCriticalError,
	}, nil
}

func (c *Checkpoints) ReloadConf(cfg Config) {
	c.log.Info("reloading configuration")
	if c.log.GetLevel() != cfg.Level.Get() {
		c.log.Info("updating log level",
			logging.String("old", c.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		c.log.SetLevel(cfg.Level.Get())
	}

	c.Config = cfg
}

func (c *Checkpoints) Close() error {
	return c.badger.Close()
}

func (c *Checkpoints) GetAll() ([]*eventspb.CheckpointEvent, error) {
	checkpoints := []*eventspb.CheckpointEvent{}
	bufs := [][]byte{}
	err := c.badger.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			buf, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			bufs = append(bufs, buf)
		}
		return nil
	})

	if err != nil {
		c.log.Error("unable to get all checkpoints", logging.Error(err))
		return nil, err
	}

	for _, buf := range bufs {
		cp := eventspb.CheckpointEvent{}
		err := proto.Unmarshal(buf, &cp)
		if err != nil {
			c.log.Error("unable to unmarshal checkpoint from badger store",
				logging.Error(err),
			)
			return nil, err
		}
		checkpoints = append(checkpoints, &cp)
	}

	return checkpoints, nil
}

func (c *Checkpoints) Save(cp *eventspb.CheckpointEvent) error {
	buf, err := proto.Marshal(cp)
	if err != nil {
		c.log.Error("unable to marshal checkpoint",
			logging.String("checkpoint-hash", cp.Hash),
			logging.Error(err),
		)
		return err
	}
	cpKey := c.badger.checkpointKey(cp.Hash, cp.BlockHash, cp.BlockHeight)
	err = c.badger.db.Update(func(txn *badger.Txn) error {
		return txn.Set(cpKey, buf)
	})
	if err != nil {
		c.log.Error("unable to save market in badger",
			logging.Error(err),
			logging.String("checkpoint-hash", cp.Hash),
		)
		c.onCriticalError()
	}

	return err
}
