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

package oracles

import (
	"context"
	"errors"
	"sort"
	"sync"

	"code.vegaprotocol.io/data-node/subscribers"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
)

var (
	ErrNoOracleSpecForID = errors.New("no oracle spec for ID")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_store_mock.go -package mocks code.vegaprotocol.io/data-node/oracles OracleSpecStore
type OracleSpecStore interface {
	Post(*oraclespb.OracleSpec) error
	GetByID(string) (*oraclespb.OracleSpec, error)
	List() ([]*oraclespb.OracleSpec, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_store_mock.go -package mocks code.vegaprotocol.io/data-node/oracles OracleDataStore
type OracleDataStore interface {
	Post(*oraclespb.OracleData) error
	GetBySpecID(string) (*oraclespb.OracleData, error)
	List() ([]*oraclespb.OracleData, error)
}

type OracleSpecEvent interface {
	events.Event
	OracleSpec() oraclespb.OracleSpec
}

type OracleDataEvent interface {
	events.Event
	OracleData() oraclespb.OracleData
}

type Service struct {
	*subscribers.Base

	specs map[string]oraclespb.OracleSpec

	// matchedOracleDataBySpecId indexes the matched oracle data by spec ID.
	// This is used to retrieve all the oracle data that matched an oracle spec.
	matchedOracleDataBySpecId map[string][]oraclespb.OracleData

	// seenOracleData keep track of all the oracle data a node has seen.
	seenOracleData []oraclespb.OracleData

	mu     sync.RWMutex
	specCh chan oraclespb.OracleSpec
	dataCh chan oraclespb.OracleData
}

func NewService(ctx context.Context) *Service {
	svc := &Service{
		Base:                      subscribers.NewBase(ctx, 10, true),
		specs:                     map[string]oraclespb.OracleSpec{},
		matchedOracleDataBySpecId: map[string][]oraclespb.OracleData{},
		seenOracleData:            []oraclespb.OracleData{},
		specCh:                    make(chan oraclespb.OracleSpec, 100),
		dataCh:                    make(chan oraclespb.OracleData, 100),
	}

	go svc.consume()
	return svc
}

func (s *Service) Push(events ...events.Event) {
	for _, e := range events {
		select {
		case <-s.Closed():
			return
		default:
			if wse, ok := e.(OracleSpecEvent); ok {
				s.specCh <- wse.OracleSpec()
			} else if wse, ok := e.(OracleDataEvent); ok {
				s.dataCh <- wse.OracleData()
			}
		}
	}
}

func (s *Service) GetSpecByID(id string) (oraclespb.OracleSpec, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	spec, ok := s.specs[id]
	if !ok {
		return oraclespb.OracleSpec{}, ErrNoOracleSpecForID
	}
	return spec, nil
}

func (s *Service) ListOracleSpecs(pagination protoapi.Pagination) []oraclespb.OracleSpec {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]oraclespb.OracleSpec, 0, len(s.specs))
	for _, spec := range s.specs {
		out = append(out, spec)
	}
	return paginateOracleSpecs(out, pagination)
}

func (s *Service) GetOracleDataBySpecID(id string, pagination protoapi.Pagination) ([]oraclespb.OracleData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sentDataList, ok := s.matchedOracleDataBySpecId[id]
	if !ok {
		return []oraclespb.OracleData{}, ErrNoOracleSpecForID
	}
	out := make([]oraclespb.OracleData, 0, len(sentDataList))
	out = append(out, sentDataList...)
	return paginateOracleData(out, pagination), nil
}

func (s *Service) ListOracleData(pagination protoapi.Pagination) []oraclespb.OracleData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]oraclespb.OracleData, 0, len(s.seenOracleData))
	for _, d := range s.seenOracleData {
		out = append(out, d)
	}
	return paginateOracleData(out, pagination)
}

func (*Service) Types() []events.Type {
	return []events.Type{
		events.OracleSpecEvent,
		events.OracleDataEvent,
	}
}

func (s *Service) consume() {
	defer func() { close(s.specCh) }()
	defer func() { close(s.dataCh) }()
	for {
		select {
		case <-s.Closed():
			return
		case spec, ok := <-s.specCh:
			if !ok {
				// cleanup base
				s.Halt()
				// channel is closed
				return
			}
			s.mu.Lock()
			s.specs[spec.Id] = spec
			s.mu.Unlock()
		case data, ok := <-s.dataCh:
			if !ok {
				// cleanup base
				s.Halt()
				// channel is closed
				return
			}
			s.saveOracleData(data)
		}

	}
}

func (s *Service) saveOracleData(data oraclespb.OracleData) {
	s.mu.Lock()

	s.seenOracleData = append(s.seenOracleData, data)

	for _, specID := range data.MatchedSpecIds {
		oracleDataList, ok := s.matchedOracleDataBySpecId[specID]
		if !ok {
			oracleDataList = []oraclespb.OracleData{}
		}
		s.matchedOracleDataBySpecId[specID] = append(oracleDataList, data)
	}

	s.mu.Unlock()
}

// paginateOracleData paginates oracle data, sorted by broadcast time
func paginateOracleData(data []oraclespb.OracleData, pagination protoapi.Pagination) []oraclespb.OracleData {
	length := uint64(len(data))
	start := uint64(0)
	end := length

	sortFn := func(i, j int) bool { return data[i].BroadcastAt < data[j].BroadcastAt }
	if pagination.Descending {
		sortFn = func(i, j int) bool { return data[i].BroadcastAt > data[j].BroadcastAt }
	}

	sort.SliceStable(data, sortFn)
	start = pagination.Skip
	if pagination.Limit != 0 {
		end = pagination.Skip + pagination.Limit
	}

	min := func(x, y uint64) uint64 {
		if y < x {
			return y
		}
		return x
	}

	start = min(start, length)
	end = min(end, length)
	return data[start:end]
}

// paginateOracleSpecs paginates oracle specs, sorted by creation time
func paginateOracleSpecs(specs []oraclespb.OracleSpec, pagination protoapi.Pagination) []oraclespb.OracleSpec {
	length := uint64(len(specs))
	start := uint64(0)
	end := length

	sortFn := func(i, j int) bool { return specs[i].CreatedAt < specs[j].CreatedAt }
	if pagination.Descending {
		sortFn = func(i, j int) bool { return specs[i].CreatedAt > specs[j].CreatedAt }
	}

	sort.SliceStable(specs, sortFn)
	start = pagination.Skip
	if pagination.Limit != 0 {
		end = pagination.Skip + pagination.Limit
	}

	min := func(x, y uint64) uint64 {
		if y < x {
			return y
		}
		return x
	}

	start = min(start, length)
	end = min(end, length)
	return specs[start:end]
}
