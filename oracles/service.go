package oracles

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
	"code.vegaprotocol.io/vega/subscribers"

	"github.com/pkg/errors"
)

var (
	ErrNoOracleSpecForID = errors.New("no oracle spec for ID")
)

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

	specs  map[string]oraclespb.OracleSpec
	data   map[string][]oraclespb.OracleData
	mu     sync.RWMutex
	specCh chan oraclespb.OracleSpec
	dataCh chan oraclespb.OracleData
}

func NewService(ctx context.Context) *Service {
	svc := &Service{
		Base:   subscribers.NewBase(ctx, 10, true),
		specs:  map[string]oraclespb.OracleSpec{},
		data:   map[string][]oraclespb.OracleData{},
		specCh: make(chan oraclespb.OracleSpec, 100),
		dataCh: make(chan oraclespb.OracleData, 100),
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
	for _, specID := range data.MatchedSpecIds {
		oracleDataList, ok := s.data[specID]
		if !ok {
			oracleDataList = []oraclespb.OracleData{}
		}
		s.data[specID] = append(oracleDataList, data)
	}
	s.mu.Unlock()
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

func (s *Service) GetSpecs() []oraclespb.OracleSpec {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]oraclespb.OracleSpec, 0, len(s.specs))
	for _, spec := range s.specs {
		out = append(out, spec)
	}
	return out
}

func (s *Service) GetOracleDataBySpecID(id string) ([]oraclespb.OracleData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sentDataList, ok := s.data[id]
	if !ok {
		return []oraclespb.OracleData{}, ErrNoOracleSpecForID
	}
	out := make([]oraclespb.OracleData, 0, len(sentDataList))
	out = append(out, sentDataList...)
	return out, nil
}

func (*Service) Types() []events.Type {
	return []events.Type{
		events.OracleSpecEvent,
		events.OracleDataEvent,
	}
}
