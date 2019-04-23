package blockchain

import (
	"fmt"
	"sync"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/pkg/errors"
)

type Service interface {
	Begin() error
	Commit() error

	SubmitOrder(order *types.Order) error
	CancelOrder(order *types.Order) error
	AmendOrder(order *types.OrderAmendment) error
	ValidateOrder(order *types.Order) error
	ReloadConf(conf Config)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/service_time_mock.go -package mocks code.vegaprotocol.io/vega/internal/blockchain ServiceTime
type ServiceTime interface {
	GetTimeNow() (time.Time, error)
	GetTimeLastBatch() (time.Time, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/service_execution_engine_mock.go -package mocks code.vegaprotocol.io/vega/internal/blockchain ServiceExecutionEngine
type ServiceExecutionEngine interface {
	SubmitOrder(order *types.Order) (*types.OrderConfirmation, error)
	CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error)
	AmendOrder(order *types.OrderAmendment) (*types.OrderConfirmation, error)
	Generate() error
}

type abciService struct {
	Config

	cfgMu             sync.Mutex
	log               *logging.Logger
	stats             *Stats
	time              ServiceTime
	execution         ServiceExecutionEngine
	previousTimestamp time.Time
	currentTimestamp  time.Time

	ordersInBatchLengths []int
	currentOrdersInBatch int
	currentTradesInBatch int
	totalBatches         uint64
	totalOrders          uint64
	totalTrades          uint64
}

func NewService(log *logging.Logger, conf Config, stats *Stats, ex ServiceExecutionEngine, timeService ServiceTime) Service {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	return &abciService{
		log:       log,
		Config:    conf,
		stats:     stats,
		execution: ex,
		time:      timeService,
	}
}

func (s *abciService) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.cfgMu.Lock()
	s.Config = cfg
	s.cfgMu.Unlock()
}

func (s *abciService) Begin() error {
	s.log.Debug("ABCI service BEGIN starting")

	// Load the latest consensus block time
	currentTime, err := s.time.GetTimeNow()
	if err != nil {
		return err
	}

	previousTime, err := s.time.GetTimeLastBatch()
	if err != nil {
		return err
	}

	s.currentTimestamp = currentTime
	s.previousTimestamp = previousTime

	s.log.Debug("ABCI service BEGIN completed",
		logging.Int64("current-timestamp", s.currentTimestamp.UnixNano()),
		logging.Int64("previous-timestamp", s.previousTimestamp.UnixNano()),
		logging.String("current-datetime", vegatime.Format(s.currentTimestamp)),
		logging.String("previous-datetime", vegatime.Format(s.previousTimestamp)),
	)

	return nil
}

func (s *abciService) ValidateOrder(order *types.Order) error {
	s.log.Debug("ABCI service validating order", logging.Order(*order))

	return nil
}

func (s *abciService) Commit() error {
	s.log.Debug("ABCI service COMMIT starting")
	s.setBatchStats()

	// Call out to run any data generation in the stores etc
	err := s.execution.Generate()
	if err != nil {
		return errors.Wrap(err, "Failure generating data in execution engine (commit)")
	}

	s.log.Debug("ABCI service COMMIT completed")
	return nil
}

func (s *abciService) SubmitOrder(order *types.Order) error {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.stats.addTotalCreateOrder(1)
	if s.LogOrderSubmitDebug {
		s.log.Debug("Blockchain service received a SUBMIT ORDER request", logging.Order(*order))
	}

	order.Id = fmt.Sprintf("V%010d-%010d", s.totalBatches, s.totalOrders)
	order.CreatedAt = s.currentTimestamp.UnixNano()

	// Submit the create order request to the execution engine
	confirmationMessage, errorMessage := s.execution.SubmitOrder(order)
	if confirmationMessage != nil {
		if s.LogOrderSubmitDebug {
			s.log.Debug("Order confirmed",
				logging.Order(*order),
				logging.OrderWithTag(*confirmationMessage.Order, "aggressive-order"),
				logging.String("passive-trades", fmt.Sprintf("%+v", confirmationMessage.Trades)),
				logging.String("passive-orders", fmt.Sprintf("%+v", confirmationMessage.PassiveOrdersAffected)))

			s.currentTradesInBatch += len(confirmationMessage.Trades)
			s.totalTrades += uint64(s.currentTradesInBatch)
		}
		s.stats.addTotalOrders(1)
		s.stats.addTotalTrades(uint64(len(confirmationMessage.Trades)))

		s.currentOrdersInBatch++
	}

	// increment total orders, even for failures so current ID strategy is valid.
	s.totalOrders++

	if errorMessage != nil {
		s.log.Error("error message on creating order",
			logging.Order(*order),
			logging.Error(errorMessage))
		return errorMessage
	}

	return nil
}

func (s *abciService) CancelOrder(order *types.Order) error {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.stats.addTotalCancelOrder(1)
	if s.LogOrderCancelDebug {
		s.log.Debug("Blockchain service received a CANCEL ORDER request", logging.Order(*order))
	}

	// Submit the cancel new order request to the Vega trading core
	cancellationMessage, errorMessage := s.execution.CancelOrder(order)
	if cancellationMessage != nil {
		if s.LogOrderCancelDebug {
			s.log.Debug("Order cancelled", logging.Order(*cancellationMessage.Order))
		}
	}
	if errorMessage != nil {
		s.log.Error("error message on cancelling order",
			logging.Order(*order),
			logging.Error(errorMessage))
		return errorMessage
	}

	return nil
}

func (s *abciService) AmendOrder(order *types.OrderAmendment) error {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.stats.addTotalAmendOrder(1)
	if s.LogOrderAmendDebug {
		s.log.Debug("Blockchain service received a AMEND ORDER request",
			logging.String("order", order.String()))
	}

	// Submit the Amendment new order request to the Vega trading core
	confirmationMessage, errorMessage := s.execution.AmendOrder(order)
	if confirmationMessage != nil {
		if s.LogOrderAmendDebug {
			s.log.Debug("Order amended", logging.String("order", order.String()))
		}
	}
	if errorMessage != nil {
		s.log.Error("error message on amending order",
			logging.String("order", order.String()),
			logging.Error(errorMessage))
		return errorMessage
	}

	return nil
}

func (s *abciService) setBatchStats() {
	s.totalBatches++
	s.stats.totalOrdersLastBatch = s.currentOrdersInBatch
	s.stats.totalTradesLastBatch = s.currentTradesInBatch

	// Calculate total orders per batch (per block) average
	if s.currentOrdersInBatch > 0 {
		if s.ordersInBatchLengths == nil {
			s.ordersInBatchLengths = make([]int, 0)
		}
		s.ordersInBatchLengths = append(s.ordersInBatchLengths, s.currentOrdersInBatch)
		lenOrdersInBatch := len(s.ordersInBatchLengths)
		if lenOrdersInBatch > 0 {
			totalOrders := 0
			for _, itx := range s.ordersInBatchLengths {
				totalOrders += itx
			}
			s.stats.averageOrdersPerBatch = totalOrders / lenOrdersInBatch

			// MAX sample size for avg calculation is 5000
			if lenOrdersInBatch == 5000 {
				s.ordersInBatchLengths = nil
			}
		}
	}

	blockDuration := time.Duration(s.currentTimestamp.UnixNano() - s.previousTimestamp.UnixNano()).Seconds()
	s.stats.setOrdersPerSecond(uint64(float64(s.currentOrdersInBatch) / blockDuration))
	s.stats.setTradesPerSecond(uint64(float64(s.currentTradesInBatch) / blockDuration))
	s.log.Info("blockduration", logging.Float64("dur", blockDuration))

	s.log.Debug("Blockchain service batch stats",
		logging.Uint64("total-batches", s.totalBatches),
		logging.Int("avg-orders-batch", s.stats.averageOrdersPerBatch),
		logging.Uint64("orders-per-sec", s.stats.OrdersPerSecond()),
		logging.Uint64("trades-per-sec", s.stats.TradesPerSecond()),
	)

	s.currentOrdersInBatch = 0
	s.currentTradesInBatch = 0
}
