package storage

import (
	"sort"

	"code.vegaprotocol.io/data-node/logging"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"

	"github.com/dgraph-io/badger/v2"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type Checkpoints struct {
	Config
	badger          *badgerStore
	log             *logging.Logger
	onCriticalError func()
}

func NewCheckpoints(log *logging.Logger, c Config, onCriticalError func()) (*Checkpoints, error) {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	if err := InitStoreDirectory(c.CheckpointsDirPath); err != nil {
		return nil, errors.Wrap(err, "error on init badger database for checkpoints storage")
	}
	db, err := newBadgerStore(getOptionsFromConfig(c.Checkpoints, c.CheckpointsDirPath, log))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for checkpoints storage")
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

func (c *Checkpoints) GetAll() ([]*protoapi.Checkpoint, error) {
	checkpoints := []*protoapi.Checkpoint{}
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
		cp := protoapi.Checkpoint{}
		err := proto.Unmarshal(buf, &cp)
		if err != nil {
			c.log.Error("unable to unmarshal checkpoint from badger store",
				logging.Error(err),
			)
			return nil, err
		}
		checkpoints = append(checkpoints, &cp)
	}

	// default sort by block height, descending
	sort.SliceStable(checkpoints, func(i, j int) bool {
		return checkpoints[i].AtBlock > checkpoints[j].AtBlock
	})

	return checkpoints, nil
}

func (c *Checkpoints) Save(cp *protoapi.Checkpoint) error {
	buf, err := proto.Marshal(cp)
	if err != nil {
		c.log.Error("unable to marshal checkpoint",
			logging.String("checkpoint-hash", cp.Hash),
			logging.Error(err),
		)
		return err
	}
	cpKey := c.badger.checkpointKey(cp.Hash, cp.BlockHash, cp.AtBlock)
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
