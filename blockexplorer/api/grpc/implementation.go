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

	"code.vegaprotocol.io/vega/blockexplorer/entities"
	"code.vegaprotocol.io/vega/blockexplorer/store"
	"code.vegaprotocol.io/vega/logging"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer"
)

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

func (b *blockExplorerAPI) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	var before, after *entities.TxCursor

	limit := b.MaxPageSizeDefault
	if req.Limit > 0 {
		limit = req.Limit
	}

	if req.Before != nil {
		cursor, err := entities.TxCursorFromString(*req.Before)
		if err != nil {
			return nil, err
		}
		before = &cursor
	}

	if req.After != nil {
		cursor, err := entities.TxCursorFromString(*req.After)
		if err != nil {
			return nil, err
		}
		after = &cursor
	}

	transactions, err := b.store.ListTransactions(ctx, req.Filters, limit, before, after)
	if err != nil {
		return nil, err
	}

	resp := pb.ListTransactionsResponse{
		Transactions: transactions,
	}

	return &resp, nil
}
