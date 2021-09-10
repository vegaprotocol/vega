package delegations

import (
	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/delegation_store_mock.go -package mocks code.vegaprotocol.io/data-node/delegations DelegationStore
type DelegationStore interface {
	GetAllDelegations() ([]*pb.Delegation, error)
	GetAllDelegationsOnEpoch(epochSeq string) ([]*pb.Delegation, error)
	GetPartyDelegations(party string) ([]*pb.Delegation, error)
	GetPartyDelegationsOnEpoch(party string, epochSeq string) ([]*pb.Delegation, error)
	GetPartyNodeDelegations(party string, node string) ([]*pb.Delegation, error)
	GetPartyNodeDelegationsOnEpoch(party string, node string, epochSeq string) ([]*pb.Delegation, error)
	GetNodeDelegations(nodeID string) ([]*pb.Delegation, error)
	GetNodeDelegationsOnEpoch(nodeID string, epochSeq string) ([]*pb.Delegation, error)
}

// Service represent the epoch service
type Service struct {
	Config
	log             *logging.Logger
	delegationStore DelegationStore
}

// NewService creates an validators service with the necessary dependencies
func NewService(
	log *logging.Logger,
	config Config,
	delegationStore DelegationStore,
) *Service {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Service{
		log:             log,
		Config:          config,
		delegationStore: delegationStore,
	}
}

// ReloadConf update the market service internal configuration
func (s *Service) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.Config = cfg
}

func (s *Service) GetAllDelegations() ([]*pb.Delegation, error) {
	return s.delegationStore.GetAllDelegations()
}
func (s *Service) GetAllDelegationsOnEpoch(epochSeq string) ([]*pb.Delegation, error) {
	return s.delegationStore.GetAllDelegationsOnEpoch(epochSeq)
}
func (s *Service) GetPartyDelegations(party string) ([]*pb.Delegation, error) {
	return s.delegationStore.GetPartyDelegations(party)
}
func (s *Service) GetPartyDelegationsOnEpoch(party string, epochSeq string) ([]*pb.Delegation, error) {
	return s.delegationStore.GetPartyDelegationsOnEpoch(party, epochSeq)
}
func (s *Service) GetPartyNodeDelegations(party string, node string) ([]*pb.Delegation, error) {
	return s.delegationStore.GetPartyNodeDelegations(party, node)
}
func (s *Service) GetPartyNodeDelegationsOnEpoch(party string, node string, epochSeq string) ([]*pb.Delegation, error) {
	return s.delegationStore.GetPartyNodeDelegationsOnEpoch(party, node, epochSeq)
}
func (s *Service) GetNodeDelegations(nodeID string) ([]*pb.Delegation, error) {
	return s.delegationStore.GetNodeDelegations(nodeID)
}
func (s *Service) GetNodeDelegationsOnEpoch(nodeID string, epochSeq string) ([]*pb.Delegation, error) {
	return s.delegationStore.GetNodeDelegationsOnEpoch(nodeID, epochSeq)
}
