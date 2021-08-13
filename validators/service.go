package validators

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"
)

// ValidatorStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/validators_store_mock.go -package mocks code.vegaprotocol.io/data-node/validators ValidatorsStore
type ValidatorsStore interface {
	GetByID(name string) (*pb.Market, error)
	GetAll() ([]*pb.Market, error)
}

// Svc represent the market service
type Svc struct {
	Config
	log             *logging.Logger
	validatorsStore ValidatorsStore
}

// NewService creates an validators service with the necessary dependencies
func NewService(
	log *logging.Logger,
	config Config,
	validatorsStore ValidatorsStore,
) (*Svc, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		log:             log,
		Config:          config,
		validatorsStore: validatorsStore,
	}, nil
}

// ReloadConf update the market service internal configuration
func (s *Svc) ReloadConf(cfg Config) {
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

func (s *Svc) GetNodeData(ctx context.Context) (*pb.NodeData, error) {
	nd := &pb.NodeData{
		StakedTotal:     "10",
		TotalNodes:      20,
		InactiveNodes:   10,
		ValidatingNodes: 10,
		AverageFee:      3.14,
		Uptime:          float32(time.Now().Day()),
	}

	return nd, nil
}

func (s *Svc) GetNodes(ctx context.Context) ([]*pb.Node, error) {
	return nil, nil
}

func (s *Svc) GetNodeByID(ctx context.Context, id string) (*pb.Node, error) {
	return nil, nil
}

func (s *Svc) GetEpochByID(ctx context.Context, id uint64) (*pb.Epoch, error) {
	return nil, nil
}

func (s *Svc) GetEpoch(ctx context.Context) (*pb.Epoch, error) {
	return nil, nil
}
