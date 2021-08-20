package nodes

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"
)

// NodeStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/validators_store_mock.go -package mocks code.vegaprotocol.io/data-node/validators NodeStore
type NodeStore interface {
	GetByID(id string) (*pb.Node, error)
	GetAll() []*pb.Node
	GetTotalNodesNumber() int
	GetValidatingNodesNumber() int
	GetStakedTotal() string
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/validators_store_mock.go -package mocks code.vegaprotocol.io/data-node/validators EpochStore
type EpochStore interface {
	GetTotalNodesUptime() time.Duration
	GetEpochByID(id string) (*pb.Epoch, error)
	GetEpoch() (*pb.Epoch, error)
}

// Service represent the node service
type Service struct {
	Config
	log        *logging.Logger
	nodeStore  NodeStore
	epochStore EpochStore
}

// NewService creates an validators service with the necessary dependencies
func NewService(
	log *logging.Logger,
	config Config,
	nodeStore NodeStore,
	epochStore EpochStore,
) *Service {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Service{
		log:        log,
		Config:     config,
		nodeStore:  nodeStore,
		epochStore: epochStore,
	}
}

// ReloadConf update the node service internal configuration
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

func (s *Service) GetNodeData(ctx context.Context) (*pb.NodeData, error) {
	return &pb.NodeData{
		StakedTotal:     s.nodeStore.GetStakedTotal(),
		TotalNodes:      uint32(s.nodeStore.GetTotalNodesNumber()),
		ValidatingNodes: uint32(s.nodeStore.GetValidatingNodesNumber()),
		Uptime:          float32(s.epochStore.GetTotalNodesUptime().Minutes()),
	}, nil
}

func (s *Service) GetNodes(ctx context.Context) ([]*pb.Node, error) {
	return s.nodeStore.GetAll(), nil
}

func (s *Service) GetNodeByID(ctx context.Context, id string) (*pb.Node, error) {
	return s.nodeStore.GetByID(id)
}
