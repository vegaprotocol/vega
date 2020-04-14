package blockchain

import (
	"fmt"
	"sync"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/pkg/errors"
)

// ServiceTime ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/service_time_mock.go -package mocks code.vegaprotocol.io/vega/blockchain ServiceTime
type ServiceTime interface {
	GetTimeNow() (time.Time, error)
	GetTimeLastBatch() (time.Time, error)
}

// ServiceExecutionEngine ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/service_execution_engine_mock.go -package mocks code.vegaprotocol.io/vega/blockchain ServiceExecutionEngine
type ServiceExecutionEngine interface {
	SubmitOrder(order *types.Order) (*types.OrderConfirmation, error)
	CancelOrder(order *types.OrderCancellation) (*types.OrderCancellationConfirmation, error)
	AmendOrder(order *types.OrderAmendment) (*types.OrderConfirmation, error)
	NotifyTraderAccount(notif *types.NotifyTraderAccount) error
	Withdraw(*types.Withdraw) error
	Generate() error
	SubmitProposal(proposal *types.Proposal) error
	VoteOnProposal(vote *types.Vote) error
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

// newService instantiate a new blockchain service
func newService(log *logging.Logger, conf Config, stats *Stats, ex ServiceExecutionEngine, timeService ServiceTime) *abciService {
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

	if s.log.GetLevel() == logging.DebugLevel {
		s.log.Debug("ABCI service BEGIN completed",
			logging.Int64("current-timestamp", s.currentTimestamp.UnixNano()),
			logging.Int64("previous-timestamp", s.previousTimestamp.UnixNano()),
			logging.String("current-datetime", vegatime.Format(s.currentTimestamp)),
			logging.String("previous-datetime", vegatime.Format(s.previousTimestamp)),
		)
	}

	return nil
}

func (s *abciService) ValidateOrder(order *types.Order) error {
	if s.log.GetLevel() == logging.DebugLevel {
		s.log.Debug("ABCI service validating order", logging.Order(*order))
	}

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

func (s *abciService) NotifyTraderAccount(notif *types.NotifyTraderAccount) error {
	return s.execution.NotifyTraderAccount(notif)
}

func (s *abciService) Withdraw(notif *types.Withdraw) error {
	return s.execution.Withdraw(notif)
}

func (s *abciService) SubmitOrder(order *types.Order) error {
	s.stats.addTotalCreateOrder(1)
	if s.log.GetLevel() == logging.DebugLevel {
		s.log.Debug("Blockchain service received a SUBMIT ORDER request", logging.Order(*order))
	}

	order.CreatedAt = s.currentTimestamp.UnixNano()

	// Submit the create order request to the execution engine
	confirmationMessage, errorMessage := s.execution.SubmitOrder(order)
	if confirmationMessage != nil {

		if s.log.GetLevel() == logging.DebugLevel {
			s.log.Debug("Order confirmed",
				logging.Order(*order),
				logging.OrderWithTag(*confirmationMessage.Order, "aggressive-order"),
				logging.String("passive-trades", fmt.Sprintf("%+v", confirmationMessage.Trades)),
				logging.String("passive-orders", fmt.Sprintf("%+v", confirmationMessage.PassiveOrdersAffected)))
		}

		s.currentTradesInBatch += len(confirmationMessage.Trades)
		s.totalTrades += uint64(s.currentTradesInBatch)
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

func (s *abciService) CancelOrder(order *types.OrderCancellation) error {
	s.stats.addTotalCancelOrder(1)
	if s.log.GetLevel() == logging.DebugLevel {
		s.log.Debug("Blockchain service received a CANCEL ORDER request", logging.String("order-id", order.OrderID))
	}

	// Submit the cancel new order request to the Vega trading core
	cancellationMessage, errorMessage := s.execution.CancelOrder(order)
	if errorMessage != nil {
		s.log.Error("error message on cancelling order",
			logging.String("order-id", order.OrderID),
			logging.Error(errorMessage))
		return errorMessage
	}
	if s.LogOrderCancelDebug {
		s.log.Debug("Order cancelled", logging.Order(*cancellationMessage.Order))
	}

	return nil
}

func (s *abciService) AmendOrder(order *types.OrderAmendment) error {
	s.stats.addTotalAmendOrder(1)
	s.log.Debug("Blockchain service received a AMEND ORDER request",
		logging.String("order", order.String()))

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
	if blockDuration <= 0.0 {
		// Timestamps are inaccurate just after startup (#233).
		s.stats.setOrdersPerSecond(0)
		s.stats.setTradesPerSecond(0)
		s.stats.setBlockDuration(0)
	} else {
		s.stats.setOrdersPerSecond(uint64(float64(s.currentOrdersInBatch) / blockDuration))
		s.stats.setTradesPerSecond(uint64(float64(s.currentTradesInBatch) / blockDuration))
		blockDurationNano := blockDuration * float64(time.Second.Nanoseconds())
		s.stats.setBlockDuration(uint64(blockDurationNano))
	}

	s.log.Debug("Blockchain service batch stats",
		logging.Int64("previousTimestamp", s.previousTimestamp.UnixNano()),
		logging.Int64("currentTimestamp", s.currentTimestamp.UnixNano()),
		logging.Float64("duration", blockDuration),
		logging.Int("currentOrdersInBatch", s.currentOrdersInBatch),
		logging.Int("currentTradesInBatch", s.currentTradesInBatch),
		logging.Uint64("total-batches", s.totalBatches),
		logging.Int("avg-orders-batch", s.stats.averageOrdersPerBatch),
		logging.Uint64("orders-per-sec", s.stats.OrdersPerSecond()),
		logging.Uint64("trades-per-sec", s.stats.TradesPerSecond()),
	)

	s.currentOrdersInBatch = 0
	s.currentTradesInBatch = 0
}

func (s *abciService) SubmitProposal(proposal *types.Proposal) error {
	if s.log.GetLevel() == logging.DebugLevel {
		s.log.Debug("Blockchain service received a SUBMIT PROPOSAL request", logging.Proposal(*proposal))
	}
	proposal.Timestamp = s.currentTimestamp.UnixNano()
	err := s.execution.SubmitProposal(proposal)
	return err
}

func (s *abciService) VoteOnProposal(vote *types.Vote) error {
	if s.log.GetLevel() == logging.DebugLevel {
		s.log.Debug("Blockchain service received a VOTE ON PROPOSAL request", logging.Vote(*vote))
	}
	err := s.execution.VoteOnProposal(vote)
	return err
}
