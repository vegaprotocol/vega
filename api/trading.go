package api

import (
	"context"
	"encoding/hex"
	"time"

	"code.vegaprotocol.io/go-wallet/crypto"
	protoapi "code.vegaprotocol.io/protos/vega/api"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/types"

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
	SubmitTransaction(ctx context.Context, bundle *types.SignedBundle, ty protoapi.SubmitTransactionRequest_Type) error
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

// value receiver is important, config can be updated, this avoids data race
func (s tradingService) validateSubmitTx(ty protoapi.SubmitTransactionRequest_Type) (protoapi.SubmitTransactionRequest_Type, error) {
	// ensure this is a known value for the type
	if _, ok := protoapi.SubmitTransactionRequest_Type_name[int32(ty)]; !ok {
		return protoapi.SubmitTransactionRequest_TYPE_UNSPECIFIED, ErrUnknownSubmitTxRequestType
	}

	switch ty {
	// FIXME(jeremy): in order to keep compatibility with existing clients
	// we allow no submiting the Type field, and default to old behaviour
	case protoapi.SubmitTransactionRequest_TYPE_UNSPECIFIED:
		ty = protoapi.SubmitTransactionRequest_TYPE_ASYNC
	case protoapi.SubmitTransactionRequest_TYPE_COMMIT:
		// commit is disabled?
		if s.conf.DisableTxCommit {
			return protoapi.SubmitTransactionRequest_TYPE_UNSPECIFIED, ErrSubmitTxCommitDisabled
		}
	}
	// ty is a known type, and not disabled, all good
	return ty, nil
}

func (s *tradingService) SubmitTransaction(ctx context.Context, req *protoapi.SubmitTransactionRequest) (*protoapi.SubmitTransactionResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("SubmitTransaction", startTime)
	if req == nil || req.Tx == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	_, err := s.validateSubmitTx(req.Type)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	if err := s.blockchain.SubmitTransaction(ctx, req.Tx, protoapi.SubmitTransactionRequest_TYPE_ASYNC); err != nil {
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

	return &protoapi.SubmitTransactionResponse{
		Success: true,
	}, nil
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
	if req.Evt == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	msg, err := req.Evt.PrepareToSign()
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	// verify the signature then
	err = verifySignature(s.log, msg, req.Signature, req.PubKey)
	if err != nil {
		// we try the other signature format
		msg, err = proto.Marshal(req.Evt)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
		}
		if err = verifySignature(s.log, msg, req.Signature, req.PubKey); err != nil {
			s.log.Debug("invalid tx signature", logging.String("pubkey", req.PubKey))
			return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
		}
	}

	var ok = true
	err = s.evtForwarder.Forward(ctx, req.Evt, req.PubKey)
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
