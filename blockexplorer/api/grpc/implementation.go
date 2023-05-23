// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package grpc

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/blockexplorer/entities"
	"code.vegaprotocol.io/vega/blockexplorer/store"
	"code.vegaprotocol.io/vega/logging"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"
	types "code.vegaprotocol.io/vega/protos/vega"
	"code.vegaprotocol.io/vega/version"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrNotMapped = errors.New("error not mapped")

type blockExplorerAPI struct {
	Config
	pb.UnimplementedBlockExplorerServiceServer
	store *store.Store
	log   *logging.Logger
}

func NewBlockExplorerAPI(store *store.Store, config Config, log *logging.Logger) pb.BlockExplorerServiceServer {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	be := blockExplorerAPI{
		Config: config,
		store:  store,
		log:    log.Named(namedLogger),
	}
	return &be
}

func (b *blockExplorerAPI) Info(ctx context.Context, _ *pb.InfoRequest) (*pb.InfoResponse, error) {
	return &pb.InfoResponse{
		Version:    version.Get(),
		CommitHash: version.GetCommitHash(),
	}, nil
}

func (b *blockExplorerAPI) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	transaction, err := b.store.GetTransaction(ctx, req.Hash)
	if err != nil {
		c := codes.Internal
		if errors.Is(err, store.ErrTxNotFound) {
			c = codes.NotFound
		} else if errors.Is(err, store.ErrMultipleTxFound) {
			c = codes.FailedPrecondition
		}
		return nil, apiError(c, err)
	}

	resp := pb.GetTransactionResponse{
		Transaction: transaction,
	}

	return &resp, nil
}

func (b *blockExplorerAPI) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	var before, after *entities.TxCursor

	limit := b.MaxPageSizeDefault
	if req.Limit > 0 {
		limit = req.Limit
	}

	if req.Before != nil {
		cursor, err := entities.TxCursorFromString(*req.Before)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, err)
		}
		before = &cursor
	}

	if req.After != nil {
		cursor, err := entities.TxCursorFromString(*req.After)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, err)
		}
		after = &cursor
	}

	transactions, err := b.store.ListTransactions(ctx, req.Filters, req.TxType, req.Party, limit, before, after)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	resp := pb.ListTransactionsResponse{
		Transactions: transactions,
	}

	return &resp, nil
}

// errorMap contains a mapping between errors and Vega numeric error codes.
var errorMap = map[string]int32{
	// General
	ErrNotMapped.Error():             10000,
	store.ErrTxNotFound.Error():      10001,
	store.ErrMultipleTxFound.Error(): 10002,
}

// apiError is a helper function to build the Vega specific Error Details that
// can be returned by gRPC API and therefore also REST, GraphQL will be mapped too.
// It takes a standardised grpcCode, a Vega specific apiError, and optionally one
// or more internal errors (error from the core, rather than API).
func apiError(grpcCode codes.Code, apiError error) error {
	s := status.Newf(grpcCode, "%v error", grpcCode)
	// Create the API specific error detail for error e.g. missing party ID
	detail := types.ErrorDetail{
		Message: apiError.Error(),
	}
	// Lookup the API specific error in the table, return not found/not mapped
	// if a code has not yet been added to the map, can happen if developer misses
	// a step, periodic checking/ownership of API package can keep this up to date.
	vegaCode, found := errorMap[apiError.Error()]
	if found {
		detail.Code = vegaCode
	} else {
		detail.Code = errorMap[ErrNotMapped.Error()]
	}
	// Pack the Vega domain specific errorDetails into the status returned by gRPC domain.
	s, _ = s.WithDetails(&detail)
	return s.Err()
}
