// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	var first, last uint32

	if req.Before != nil && req.After != nil && (req.First > 0 || req.Last > 0) {
		return nil, apiError(codes.InvalidArgument, errors.New("cannot use neither limits `first`, nor `last` when both cursors `before` and `after` are set"))
	}

	if req.First > 0 && req.Last > 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("cannot use both limits `first` and `last` within the same query"))
	}

	if req.Before != nil {
		cursor, err := entities.TxCursorFromString(*req.Before)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, err)
		}
		before = &cursor
		last = b.MaxPageSizeDefault
	}

	if req.After != nil {
		cursor, err := entities.TxCursorFromString(*req.After)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, err)
		}
		after = &cursor
		first = b.MaxPageSizeDefault
	}

	if before != nil && after != nil {
		// The order of the parameters may seem odd, but this is expected as we have
		// to keep in mind the natural order of the block-explorer is reverse-chronological.
		// so, given transactions 4.2, 4.1, 3.2, 3.1, 2.2, when applying the window between
		// 3.1 and 4.2, then we have to set after to 3.1 and before to 4.2.
		// So effectively, after is the start and before is the end of the set.
		if entities.AreValidCursorBoundaries(after, before) {
			return nil, apiError(codes.InvalidArgument, errors.New("cursors `before` and `after` do not create a valid window"))
		}
	}

	if req.First > 0 {
		if req.Before != nil {
			return nil, apiError(codes.InvalidArgument, errors.New("cannot use cursor `before` when using limit `first`"))
		}
		first = req.First
	} else if req.Last > 0 {
		if req.After != nil {
			return nil, apiError(codes.InvalidArgument, errors.New("cannot use cursor `after` when using limit `last`"))
		}
		last = req.Last
	}

	// Entering this condition means there is no pagination set, so it defaults
	// to listing the MaxPageSizeDefault newest transactions.
	// Note, setting limits on a cursor window is not supported.
	if !(before != nil && after != nil) && first == 0 && last == 0 {
		first = b.MaxPageSizeDefault
	}

	transactions, err := b.store.ListTransactions(ctx,
		req.Filters,
		req.CmdTypes,
		req.ExcludeCmdTypes,
		req.Parties,
		first,
		after,
		last,
		before,
	)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &pb.ListTransactionsResponse{
		Transactions: transactions,
	}, nil
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
