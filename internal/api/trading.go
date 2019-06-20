package api

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/internal/auth"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAuthDisabled       = errors.New("auth disabled")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAuthRequired       = errors.New("auth required")
	ErrMissingOrder       = errors.New("missing order in request payload")
	ErrMissingTraderID    = errors.New("missing trader id")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_order_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api TradeOrderService
type TradeOrderService interface {
	CreateOrder(ctx context.Context, submission *types.OrderSubmission) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, cancellation *types.OrderCancellation) (*types.PendingOrder, error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (*types.PendingOrder, error)
}

type AccountService interface {
	NotifyTraderAccount(ctx context.Context, notif *types.NotifyTraderAccount) (bool, error)
}

type tradingService struct {
	log               *logging.Logger
	tradeOrderService TradeOrderService
	accountService    AccountService
	statusChecker     *monitoring.Status

	authEnabled bool
	parties     []auth.PartyInfo
	mu          sync.Mutex
}

func (s *tradingService) UpdateParties(parties []auth.PartyInfo) {
	s.mu.Lock()
	s.parties = parties
	s.mu.Unlock()
}

func (s *tradingService) validateToken(partyID string, tkn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.parties {
		if v.ID == partyID && v.Token == tkn {
			return nil
		}
	}
	return ErrInvalidCredentials
}

func (s *tradingService) SignIn(
	ctx context.Context, req *protoapi.SignInRequest,
) (*protoapi.SignInResponse, error) {
	if len(req.Id) <= 0 {
		return nil, errors.New("missing username")
	}
	if len(req.Password) <= 0 {
		return nil, errors.New("missing password")
	}

	var tkn string
	saltpass := fmt.Sprintf("vega%v", req.Password)

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.authEnabled {
		return nil, ErrAuthDisabled
	}
	for _, v := range s.parties {
		if v.ID == req.Id {
			if err := bcrypt.CompareHashAndPassword([]byte(v.Password), []byte(saltpass)); err != nil {
				s.log.Debug("invalid password",
					logging.String("user-id", v.ID),
					logging.Error(err),
				)
				return nil, ErrInvalidCredentials
			}
			tkn = v.Token
		}
	}

	if len(tkn) <= 0 {
		return nil, ErrInvalidCredentials
	}

	return &protoapi.SignInResponse{
		Token: tkn,
	}, nil
}

// CreateOrder is used to request sending an order into the VEGA platform, via consensus.
func (s *tradingService) SubmitOrder(
	ctx context.Context, req *protoapi.SubmitOrderRequest,
) (*types.PendingOrder, error) {
	if s.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return nil, ErrChainNotConnected
	}

	if req.Submission == nil {
		return nil, ErrMissingOrder
	}

	// check auth if required
	if s.authEnabled {
		if len(req.Token) <= 0 {
			s.log.Debug("missing token")
			return nil, errors.New("missing auth token")
		}
		if err := s.validateToken(req.Submission.PartyID, req.Token); err != nil {
			s.log.Debug("token error", logging.Error(err))
			return nil, err
		}
	}

	return s.tradeOrderService.CreateOrder(ctx, req.Submission)
}

// CancelOrder is used to request cancelling an order into the VEGA platform, via consensus.
func (s *tradingService) CancelOrder(
	ctx context.Context, req *protoapi.CancelOrderRequest,
) (*types.PendingOrder, error) {
	if s.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return nil, ErrChainNotConnected
	}

	if req.Cancellation == nil {
		return nil, ErrMissingOrder
	}

	// check auth if required
	if s.authEnabled {
		if len(req.Token) <= 0 {
			s.log.Debug("missing token")
			return nil, errors.New("missing auth token")
		}
		if err := s.validateToken(req.Cancellation.PartyID, req.Token); err != nil {
			s.log.Debug("token error", logging.Error(err))
			return nil, err
		}
	}

	return s.tradeOrderService.CancelOrder(ctx, req.Cancellation)
}

// AmendOrder is used to request editing an order onto the VEGA platform, via consensus.
func (s *tradingService) AmendOrder(
	ctx context.Context, req *protoapi.AmendOrderRequest,
) (*types.PendingOrder, error) {

	if req.Amendment == nil {
		return nil, ErrMissingOrder
	}

	// check auth if required
	if s.authEnabled {
		if len(req.Token) <= 0 {
			s.log.Debug("missing token")
			return nil, errors.New("missing auth token")
		}
		if err := s.validateToken(req.Amendment.PartyID, req.Token); err != nil {
			s.log.Debug("token error", logging.Error(err))
			return nil, err
		}
	}

	return s.tradeOrderService.AmendOrder(ctx, req.Amendment)
}

func (s *tradingService) NotifyTraderAccount(
	ctx context.Context, req *protoapi.NotifyTraderAccountRequest,
) (*protoapi.NotifyTraderAccountResponse, error) {
	if len(req.Notif.TraderID) <= 0 {
		return nil, ErrMissingTraderID
	}

	submitted, err := s.accountService.NotifyTraderAccount(ctx, req.Notif)
	if err != nil {
		return nil, err
	}

	return &protoapi.NotifyTraderAccountResponse{
		Submitted: submitted,
	}, nil
}
