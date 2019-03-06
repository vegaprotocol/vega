package blockchain

import (
	"fmt"
	"vega/internal/execution"
	"vega/internal/vegatime"

	types "vega/proto"

	"github.com/pkg/errors"
	"vega/internal/logging"
)

type Service interface {
	Begin() error
	Commit() error

	SubmitOrder(order *types.Order) error
	CancelOrder(order *types.Order) error
	AmendOrder(order *types.Amendment) error
	ValidateOrder(order *types.Order) error
}

type abciService struct {
	*Config
	*Stats

	time              vegatime.Service
	execution         execution.Engine
	previousTimestamp vegatime.Stamp
	currentTimestamp  vegatime.Stamp

	ordersInBatchLengths []int
	currentOrdersInBatch int
	currentTradesInBatch int
	totalBatches         uint64
	totalOrders          uint64
	totalTrades          uint64
}

func NewAbciService(conf *Config, stats *Stats, ex execution.Engine, timeService vegatime.Service) Service {
	return &abciService{Config: conf, Stats: stats, execution: ex, time: timeService}
}

func (s *abciService) Begin() error {
	s.log.Debug("ABCI service BEGIN starting")

	// Load the latest consensus block time
	epochTimeNano, _, err := s.time.GetTimeNow()
	if err != nil {
		return err
	}

	// We need to cache the last timestamp so we can distribute trades
	// in a block evenly between last timestamp and current timestamp
	if epochTimeNano > 0 {
		s.previousTimestamp = epochTimeNano
	}

	// Store the timestamp info that we receive from the blockchain provider
	s.currentTimestamp = epochTimeNano

	// Ensure we always set app.previousTimestamp it'll be 0 on the first block
	if s.previousTimestamp < 1 {
		s.previousTimestamp = epochTimeNano
	}

	// Run any processing required in execution engine, e.g. check for expired orders
	err = s.execution.Process()
	if err != nil {
		return err
	}

	s.log.Debug("ABCI service BEGIN completed",
		logging.Uint64("current-timestamp", s.currentTimestamp.Uint64()),
		logging.Uint64("previous-timestamp", s.previousTimestamp.Uint64()),
		logging.String("current-datetime", s.currentTimestamp.Rfc3339Nano()),
		logging.String("previous-datetime", s.previousTimestamp.Rfc3339Nano()),
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

	return nil
}

func (s *abciService) SubmitOrder(order *types.Order) error {
	if s.LogOrderSubmitDebug {
		s.log.Debug("Blockchain service received a SUBMIT ORDER request", logging.Order(*order))
	}

	order.Id = fmt.Sprintf("V%010d-%010d", s.totalBatches, s.totalOrders)
	order.Timestamp = s.currentTimestamp.Uint64()

	// Submit the create order request to the execution engine
	confirmationMessage, errorMessage := s.execution.SubmitOrder(order)
	if confirmationMessage != nil {
		if s.LogOrderSubmitDebug {
			s.log.Debug("Order confirmed",
				logging.Order(*order),
				logging.OrderWithTag(*confirmationMessage.Order, "aggressive-order"),
				logging.String("passive-trades", fmt.Sprintf("%+v", confirmationMessage.Trades)),
				logging.String("passive-orders", fmt.Sprintf("%+v", confirmationMessage.PassiveOrdersAffected)))

			s.totalTrades += uint64(len(confirmationMessage.Trades))
		}

		confirmationMessage.Release()
	}

	// increment total orders, even for failures so current ID strategy is valid.
	s.totalOrders++

	if errorMessage != types.OrderError_NONE {
		errorMessageString := errorMessage.String()
		s.log.Error("error message on creating order",
			logging.Order(*order),
			logging.String("error", errorMessageString))
		return errors.New(errorMessageString)
	}

	s.log.Debug("ABCI service COMMIT completed")
	return nil
}

func (s *abciService) CancelOrder(order *types.Order) error {
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
	if errorMessage != types.OrderError_NONE {
		errorMessageString := errorMessage.String()
		s.log.Error("error message on cancelling order",
			logging.Order(*order),
			logging.String("error", errorMessageString))
		return errors.New(errorMessageString)
	}

	return nil
}

func (s *abciService) AmendOrder(order *types.Amendment) error {
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
	if errorMessage != types.OrderError_NONE {
		errorMessageString := errorMessage.String()
		s.log.Error("error message on amending order",
			logging.String("order", order.String()),
			logging.String("error", errorMessageString))
		return errors.New(errorMessageString)
	}

	return nil
}

func (s *abciService) setBatchStats() {
	s.totalBatches++
	s.totalOrdersLastBatch = s.currentOrdersInBatch
	s.totalTradesLastBatch = s.currentTradesInBatch

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
			s.averageOrdersPerBatch = totalOrders / lenOrdersInBatch

			// MAX sample size for avg calculation is 5000
			if lenOrdersInBatch == 5000 {
				s.ordersInBatchLengths = nil
			}
		}
	}

	s.log.Debug("Blockchain service batch stats",
		logging.Uint64("total-batches", s.totalBatches),
		logging.Int("avg-orders-batch", s.averageOrdersPerBatch))

	s.currentOrdersInBatch = 0
	s.currentTradesInBatch = 0
}
