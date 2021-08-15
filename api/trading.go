package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"code.vegaprotocol.io/go-wallet/crypto"
	protoapi "code.vegaprotocol.io/protos/vega/api"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
)

var (
	ErrInvalidSignature           = errors.New("invalid signature")
	ErrSubmitTxCommitDisabled     = errors.New("broadcast_tx_commit is disabled")
	ErrUnknownSubmitTxRequestType = errors.New("invalid broadcast_tx type")
)

// EvtForwarder
//go:generate go run github.com/golang/mock/mockgen -destination mocks/evt_forwarder_mock.go -package mocks code.vegaprotocol.io/vega/api  EvtForwarder
type EvtForwarder interface {
	Forward(ctx context.Context, e *commandspb.ChainEvent, pk string) error
}

// Blockchain ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_mock.go -package mocks code.vegaprotocol.io/vega/api  Blockchain
type Blockchain interface {
	SubmitTransactionV2(ctx context.Context, tx *commandspb.Transaction, ty protoapi.SubmitTransactionV2Request_Type) error
}

type tradingService struct {
	log  *logging.Logger
	conf Config

	blockchain    Blockchain
	marketService MarketService
	evtForwarder  EvtForwarder

	statusChecker *monitoring.Status
}

// no need for a mutext - we only access the config through a value receiver
func (s *tradingService) updateConfig(conf Config) {
	s.conf = conf
}

func (s *tradingService) SubmitTransactionV2(ctx context.Context, req *protoapi.SubmitTransactionV2Request) (*protoapi.SubmitTransactionV2Response, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("SubmitTransactionV2", startTime)

	if req == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	if err := s.blockchain.SubmitTransactionV2(ctx, req.Tx, protoapi.SubmitTransactionV2Request_TYPE_ASYNC); err != nil {
		// This is Tendermint's specific error signature
		if _, ok := err.(interface {
			Code() uint32
			Details() string
			Error() string
		}); ok {
			s.log.Debug("unable to submit transaction", logging.Error(err))
			return nil, apiError(codes.InvalidArgument, err)
		}
		s.log.Debug("unable to submit transaction", logging.Error(err))
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.SubmitTransactionV2Response{
		Success: true,
	}, nil
}

func (s *tradingService) PropagateChainEvent(ctx context.Context, req *protoapi.PropagateChainEventRequest) (*protoapi.PropagateChainEventResponse, error) {
	if req.Event == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	// verify the signature then
	err := verifySignature(s.log, req.Event, req.Signature, req.PubKey)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("not a valid signature: %w", err))
	}

	evt := commandspb.ChainEvent{}
	err = proto.Unmarshal(req.Event, &evt)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("not a valid chain event: %w", err))
	}

	var ok = true
	err = s.evtForwarder.Forward(ctx, &evt, req.PubKey)
	if err != nil && err != evtforward.ErrEvtAlreadyExist {
		s.log.Error("unable to forward chain event",
			logging.String("pubkey", req.PubKey),
			logging.Error(err))
		if err == evtforward.ErrPubKeyNotAllowlisted {
			return nil, apiError(codes.PermissionDenied, err)
		} else {
			return nil, apiError(codes.Internal, err)
		}
	}

	return &protoapi.PropagateChainEventResponse{
		Success: ok,
	}, nil
}

func verifySignature(
	log *logging.Logger,
	message []byte,
	sig []byte,
	pubKey string,
) error {
	validator, err := crypto.NewSignatureAlgorithm(crypto.Ed25519, 1)
	if err != nil {
		if log != nil {
			log.Error("unable to instantiate new algorithm", logging.Error(err))
		}
		return err
	}

	pubKeyBytes, err := hex.DecodeString(pubKey)
	if err != nil {
		if log != nil {
			log.Error("unable to decode hexencoded ubkey", logging.Error(err))
		}
		return err
	}
	ok, err := validator.Verify(pubKeyBytes, message, sig)
	if err != nil {
		if log != nil {
			log.Error("unable to verify bundle", logging.Error(err))
		}
		return err
	}
	if !ok {
		return ErrInvalidSignature
	}
	return nil
}
