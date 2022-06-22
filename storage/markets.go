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

	cfgencoding "code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	"github.com/pkg/errors"

	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/golang/protobuf/proto"
)

var (
	ErrMarketDoNotExist = errors.New("market does not exist")
)

// Market is used for memory/RAM based markets storage.
type Market struct {
	Config

	log             *logging.Logger
	badger          *badgerStore
	onCriticalError func()
}

// NewMarkets returns a concrete implementation of MarketStore.
func NewMarkets(log *logging.Logger, home string, c Config, onCriticalError func()) (*Market, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	db, err := badger.Open(getOptionsFromConfig(c.Markets, home, log))
	if err != nil {
		return nil, fmt.Errorf("couldn't open Badger markets database: %w", err)
	}
	bs := badgerStore{db: db}
	return &Market{
		log:             log,
		Config:          c,
		badger:          &bs,
		onCriticalError: onCriticalError,
	}, nil
}

// ReloadConf update the internal conf of the market
func (m *Market) ReloadConf(cfg Config) {
	m.log.Info("reloading configuration")
	if m.log.GetLevel() != cfg.Level.Get() {
		m.log.Info("updating log level",
			logging.String("old", m.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		m.log.SetLevel(cfg.Level.Get())
	}

	m.Config = cfg
}

// Post saves a given market to the mem-store.
func (m *Market) Post(market *types.Market) error {
	buf, err := proto.Marshal(market)
	if err != nil {
		mktID := "nil"
		if market != nil {
			mktID = market.Id
		}
		m.log.Error("unable to marshal market",
			logging.Error(err),
			logging.String("market-id", mktID),
		)
		return err
	}
	marketKey := m.badger.marketKey(market.Id)
	err = m.badger.db.Update(func(txn *badger.Txn) error {
		return txn.Set(marketKey, buf)
	})
	if err != nil {
		m.log.Error("unable to save market in badger",
			logging.Error(err),
			logging.String("market-id", market.Id),
		)
		m.onCriticalError()
	}

	return err
}

// GetByID searches for the given market by id in the mem-store.
func (m *Market) GetByID(id string) (*types.Market, error) {
	market := types.Market{}
	var buf []byte
	marketKey := m.badger.marketKey(id)
	err := m.badger.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(marketKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrMarketDoNotExist
			}
			return err
		}
		// fine to use value copy here, only one ID to get
		buf, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		m.log.Error("unable to get market from badger store",
			logging.Error(err),
			logging.String("market-id", id),
		)
		return nil, err
	}

	err = proto.Unmarshal(buf, &market)
	if err != nil {
		m.log.Error("unable to unmarshal market from badger store",
			logging.Error(err),
			logging.String("market-id", id),
		)
		return nil, err
	}
	return &market, nil
}

// GetAll returns all markets in the badger store.
func (m *Market) GetAll() ([]*types.Market, error) {
	markets := []*types.Market{}
	bufs := [][]byte{}
	err := m.badger.db.View(func(txn *badger.Txn) error {
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
		m.log.Error("unable to get all markets", logging.Error(err))
		return nil, err
	}

	for _, buf := range bufs {
		mkt := types.Market{}
		err := proto.Unmarshal(buf, &mkt)
		if err != nil {
			m.log.Error("unable to unmarshal market from badger store",
				logging.Error(err),
			)
			return nil, err
		}
		markets = append(markets, &mkt)
	}

	return markets, nil
}

func (m *Market) SaveBatch(batch []types.Market) error {
	for _, v := range batch {
		if err := m.Post(&v); err != nil {
			return err
		}
	}
	return nil
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (m *Market) Close() error {
	return m.badger.db.Close()
}

// DefaultMarketStoreOptions supplies default options we use for market stores
// currently we want to load market value log and LSM tree via a MemoryMap.
// Note: markets total likely to be less than 1000 on a shard, short term.
func DefaultMarketStoreOptions() ConfigOptions {
	opts := DefaultStoreOptions()
	opts.TableLoadingMode = cfgencoding.FileLoadingMode{FileLoadingMode: options.MemoryMap}
	opts.ValueLogLoadingMode = cfgencoding.FileLoadingMode{FileLoadingMode: options.MemoryMap}
	return opts
}
