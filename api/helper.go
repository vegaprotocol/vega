package api

import (
	"context"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"google.golang.org/grpc/codes"
)

func subscriptionHelper[RespType any, ProtoType any, inType interface{ ToProto() *ProtoType }, ReqType any](
	ctx context.Context,
	name string,
	ch <-chan []inType,
	ref uint64,
	req *ReqType,
	srv interface{ Send(*RespType) error },
	log *logging.Logger,
	responseBuilder func([]*ProtoType) *RespType,

) error {
	defer metrics.StartActiveSubscriptionCountGRPC(name)()

	if log.GetLevel() == logging.DebugLevel {
		log.Debug(name+" subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	var err error
	for {
		select {
		case val, ok := <-ch:
			if !ok {
				err = ErrChannelClosed
				log.Error(name+" subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			protoMds := make([]*ProtoType, len(val))
			for i, md := range val {
				protoMds[i] = md.ToProto()
			}

			resp := responseBuilder(protoMds)

			if err := srv.Send(resp); err != nil {
				log.Error(name+" subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}

		case <-ctx.Done():
			err = ctx.Err()
			if log.GetLevel() == logging.DebugLevel {
				log.Debug(name+" subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		}

		if ch == nil {
			if log.GetLevel() == logging.DebugLevel {
				log.Debug(name+" subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}
